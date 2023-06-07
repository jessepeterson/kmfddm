package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/flow"
	httpddm "github.com/jessepeterson/kmfddm/http"
	apihttp "github.com/jessepeterson/kmfddm/http/api"
	ddmhttp "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/log/stdlogfmt"
	"github.com/jessepeterson/kmfddm/notifier"
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
		flVersion = flag.Bool("version", false, "print version")
		flStorage = flag.String("storage", "file", "storage backend")
		flDSN     = flag.String("storage-dsn", "", "storage data source name")

		flDumpStatus = flag.String("dump-status", "", "file name to dump status reports to (\"-\" for stdout)")

		flEnqueueURL = flag.String("enqueue", "", "URL of MDM server enqueue endpoint")
		flEnqueueKey = flag.String("enqueue-key", "", "MDM server enqueue API key")
		flCORSOrigin = flag.String("cors-origin", "", "CORS Origin; for browser-based API access")
		flMicro      = flag.Bool("micromdm", false, "Use MicroMDM command API calling conventions")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	if *flAPIKey == "" {
		logger.Info("msg", "empty API key; API disabled")
	}

	storage, err := storage(*flStorage, *flDSN)
	if err != nil {
		logger.Info("msg", "init storage", "name", *flStorage, "err", err)
		os.Exit(1)
	}

	nOpts := []notifier.Option{
		notifier.WithLogger(logger.With("service", "notifier")),
	}
	if *flMicro {
		nOpts = append(nOpts, notifier.WithMicroMDM())
	}
	nanoNotif, err := notifier.New(storage, *flEnqueueURL, *flEnqueueKey, nOpts...)
	if err != nil {
		logger.Info("msg", "creating notifier", "err", err)
		os.Exit(1)
	}

	mux := flow.New()

	mux.Handle("/version", httpddm.VersionHandler(version))

	mux.Handle(
		"/declaration-items",
		ddmhttp.TokensDeclarationItemsHandler(storage, false, logger.With("handler", "declaration-items")),
		"GET",
	)

	mux.Handle(
		"/tokens",
		ddmhttp.TokensDeclarationItemsHandler(storage, true, logger.With("handler", "tokens")),
		"GET",
	)

	mux.Handle(
		"/declaration/:type/:id",
		http.StripPrefix("/declaration/",
			ddmhttp.DeclarationHandler(storage, logger.With("handler", "declaration")),
		),
		"GET",
	)

	var statusHandler http.Handler = ddmhttp.StatusReportHandler(storage, logger.With("handler", "status"))
	if *flDumpStatus != "" {
		f := os.Stdout
		if *flDumpStatus != "-" {
			if f, err = os.OpenFile(*flDumpStatus, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
				logger.Info("msg", "dump status", "path", *flDumpStatus, "err", err)
				os.Exit(1)
			}
			defer f.Close()
			logger.Debug("msg", "dump status", "path", *flDumpStatus)
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
				return httpddm.BasicAuthMiddleware(h, apiUsername, *flAPIKey, apiRealm)
			})

			// declarations
			mux.Handle(
				"/v1/declarations",
				apihttp.GetDeclarationsHandler(storage, logger.With("get-declarations")),
				"GET",
			)

			mux.Handle(
				"/v1/declarations",
				apihttp.PutDeclarationHandler(storage, nanoNotif, logger.With("handler", "put-declaration")),
				"PUT",
			)

			mux.Handle(
				"/v1/declarations/:id",
				apihttp.GetDeclarationHandler(storage, logger.With("handler", "get-declaration")),
				"GET",
			)

			mux.Handle(
				"/v1/declarations/:id",
				apihttp.DeleteDeclarationHandler(storage, logger.With("handler", "delete-declaration")),
				"DELETE",
			)

			// sets
			mux.Handle(
				"/v1/sets",
				apihttp.GetSetsHandler(storage, logger.With("get-sets")),
				"GET",
			)

			// set declarations
			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.GetSetDeclarationsHandler(storage, logger.With("handler", "get-set-declarations")),
				"GET",
			)

			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.PutSetDeclarationHandler(storage, nanoNotif, logger.With("handler", "put-set-declarations")),
				"PUT",
			)

			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.DeleteSetDeclarationHandler(storage, nanoNotif, logger.With("handler", "delete-set-delcarations")),
				"DELETE",
			)

			// enrollment sets
			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.GetEnrollmentSetsHandler(storage, logger.With("handler", "get-enrollment-sets")),
				"GET",
			)

			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.PutEnrollmentSetHandler(storage, nanoNotif, logger.With("handler", "put-enrollment-sets")),
				"PUT",
			)

			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.DeleteEnrollmentSetHandler(storage, nanoNotif, logger.With("handler", "delete-enrollment-sets")),
				"DELETE",
			)

			// declarations sets
			mux.Handle(
				"/v1/declaration-sets/:id",
				apihttp.GetDeclarationSetsHandler(storage, logger.With("handler", "get-declaration-sets")),
				"GET",
			)

			// status queries
			mux.Handle(
				"/v1/declaration-status/:id",
				apihttp.GetDeclarationStatusHandler(storage, logger.With("handler", "get-declaration-status")),
				"GET",
			)

			mux.Handle(
				"/v1/status-errors/:id",
				apihttp.GetStatusErrorsHandler(storage, logger.With("handler", "get-status-errors")),
				"GET",
			)

			mux.Handle(
				"/v1/status-values/:id",
				apihttp.GetStatusValuesHandler(storage, logger.With("handler", "get-status-values")),
				"GET",
			)
		})
	}

	// init for newTraceID()
	rand.Seed(time.Now().UnixNano())

	logger.Info("msg", "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, httpddm.TraceLoggingMiddleware(mux, logger.With("handler", "log"), newTraceID))
	logs := []interface{}{"msg", "server shutdown"}
	if err != nil {
		logs = append(logs, "err", err)
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
