package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	httpddm "github.com/jessepeterson/kmfddm/http"
	apihttp "github.com/jessepeterson/kmfddm/http/api"
	ddmhttp "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/notifier"
	"github.com/jessepeterson/kmfddm/notifier/foss"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/shard"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/envflag"
	nanohttp "github.com/micromdm/nanolib/http"
	"github.com/micromdm/nanolib/http/trace"
	"github.com/micromdm/nanolib/log/stdlogfmt"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	apiUsername = "kmfddm"
	apiRealm    = "kmfddm"
)

func main() {
	var (
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flListen  = flag.String("listen", ":9002", "HTTP listen address")
		flAPIKey  = flag.String("api", "", "API key for API endpoints")
		flVersion = flag.Bool("version", false, "print version and exit")
		flStorage = flag.String("storage", "filekv", "storage backend")
		flDSN     = flag.String("storage-dsn", "", "storage data source name")
		flOptions = flag.String("storage-options", "", "storage backend options")

		flShard = flag.Bool("shard", false, "enable shard management properties declaration")

		flDumpStatus = flag.String("dump-status", "", "file name to dump status reports to (\"-\" for stdout)")

		flEnqueueURL = flag.String("enqueue", "", "URL of MDM server enqueue endpoint")
		flEnqueueKey = flag.String("enqueue-key", "", "MDM server enqueue API key")
		flCORSOrigin = flag.String("cors-origin", "", "CORS Origin; for browser-based API access")
		flMicro      = flag.Bool("micromdm", false, "Use MicroMDM command API calling conventions")
	)
	envflag.Parse("KMFDDM_", []string{"version"})

	if *flVersion {
		fmt.Println(version)
		return
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	if *flAPIKey == "" {
		logger.Info(logkeys.Message, "empty API key; API disabled")
	}

	var store allStorage
	var err error
	store, err = setupStorage(*flStorage, *flDSN, *flOptions, logger)
	if err != nil {
		logger.Info(logkeys.Message, "init storage", "name", *flStorage, logkeys.Error, err)
		os.Exit(1)
	}

	nOpts := []foss.Option{
		foss.WithLogger(logger.With("service", "notifier-foss")),
	}
	if *flMicro {
		nOpts = append(nOpts, foss.WithMicroMDM())
	}
	fossNotif, err := foss.NewFossMDM(*flEnqueueURL, *flEnqueueKey, nOpts...)
	if err != nil {
		logger.Info(logkeys.Message, "creating notifier", logkeys.Error, err)
		os.Exit(1)
	}

	var ddmStore storage.EnrollmentDeclarationStorage = store

	if *flShard {
		// compose DDM storage out of shard storage and the underlying storage
		ddmStore = storage.NewJSONAdapt(storage.NewMulti(shard.NewShardStorage(), store), hasher)
	}

	nanoNotif, err := notifier.New(fossNotif, store, notifier.WithLogger(logger.With("service", "notifier")))
	if err != nil {
		logger.Info(logkeys.Message, "creating notifier", logkeys.Error, err)
		os.Exit(1)
	}

	mux := flow.New()

	mux.Handle("/version", nanohttp.NewJSONVersionHandler(version))

	mux.Handle(
		"/declaration-items",
		ddmhttp.TokensOrDeclarationItemsHandler(ddmStore, false, logger.With(logkeys.Handler, "declaration-items")),
		"GET",
	)

	mux.Handle(
		"/tokens",
		ddmhttp.TokensOrDeclarationItemsHandler(ddmStore, true, logger.With(logkeys.Handler, "tokens")),
		"GET",
	)

	mux.Handle(
		"/declaration/:type/:id",
		http.StripPrefix("/declaration/",
			ddmhttp.DeclarationHandler(ddmStore, logger.With(logkeys.Handler, "declaration")),
		),
		"GET",
	)

	var statusHandler http.Handler = ddmhttp.StatusReportHandler(store, logger.With(logkeys.Handler, "status"))
	if *flDumpStatus != "" {
		f := os.Stdout
		if *flDumpStatus != "-" {
			if f, err = os.OpenFile(*flDumpStatus, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
				logger.Info(logkeys.Message, "dump status", "path", *flDumpStatus, logkeys.Error, err)
				os.Exit(1)
			}
			defer f.Close()
			logger.Debug(logkeys.Message, "dump status", "path", *flDumpStatus)
		}
		statusHandler = DumpHandler(statusHandler, f)
	}
	mux.Handle("/status", statusHandler, "PUT")

	if *flAPIKey != "" {
		if *flCORSOrigin != "" {
			// for middleware to work on the OPTIONS method using flow router
			// we must define a middleware on the "root" mux
			mux.Use(func(h http.Handler) http.Handler {
				return httpddm.CORSMiddleware(h, *flCORSOrigin)
			})
		}

		mux.Group(func(mux *flow.Mux) {
			mux.Use(func(h http.Handler) http.Handler {
				return nanohttp.NewSimpleBasicAuthHandler(h, apiUsername, *flAPIKey, apiRealm)
			})

			apihttp.HandleAPIv1("/v1", mux, logger, store, nanoNotif)
		})
	}

	// init for newTraceID()
	rand.Seed(time.Now().UnixNano())

	logger.Info(logkeys.Message, "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, trace.NewTraceLoggingHandler(mux, logger.With(logkeys.Handler, "log"), newTraceID))
	logs := []interface{}{logkeys.Message, "server shutdown"}
	if err != nil {
		logs = append(logs, logkeys.Error, err)
	}
	logger.Info(logs...)
}

// newTraceID generates a new HTTP trace ID for context logging.
// Currently this just makes a random string. This would be better
// served by e.g. https://github.com/oklog/ulid or something like
// https://opentelemetry.io/ someday.
func newTraceID(_ *http.Request) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func DumpHandler(next http.Handler, output io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respBytes, _ := httpddm.ReadAllAndReplaceBody(r)
		output.Write(respBytes)
		output.Write([]byte{'\n'})
		next.ServeHTTP(w, r)
	}
}
