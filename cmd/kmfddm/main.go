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
	"github.com/jessepeterson/kmfddm/log/logkeys"
	"github.com/jessepeterson/kmfddm/log/stdlogfmt"
	"github.com/jessepeterson/kmfddm/notifier"
	"github.com/jessepeterson/kmfddm/notifier/foss"
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
		flOptions = flag.String("storage-options", "", "storage backend options")

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
	nanoNotif, err := notifier.New(fossNotif, store, notifier.WithLogger(logger.With("service", "notifier")))
	if err != nil {
		logger.Info(logkeys.Message, "creating notifier", logkeys.Error, err)
		os.Exit(1)
	}

	mux := flow.New()

	mux.Handle("/version", httpddm.VersionHandler(version))

	mux.Handle(
		"/declaration-items",
		ddmhttp.TokensOrDeclarationItemsHandler(store, false, logger.With(logkeys.Handler, "declaration-items")),
		"GET",
	)

	mux.Handle(
		"/tokens",
		ddmhttp.TokensOrDeclarationItemsHandler(store, true, logger.With(logkeys.Handler, "tokens")),
		"GET",
	)

	mux.Handle(
		"/declaration/:type/:id",
		http.StripPrefix("/declaration/",
			ddmhttp.DeclarationHandler(store, logger.With(logkeys.Handler, "declaration")),
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
				return httpddm.BasicAuthMiddleware(h, apiUsername, *flAPIKey, apiRealm)
			})

			// declarations
			mux.Handle(
				"/v1/declarations",
				apihttp.GetDeclarationsHandler(store, logger.With("get-declarations")),
				"GET",
			)

			mux.Handle(
				"/v1/declarations",
				apihttp.PutDeclarationHandler(store, nanoNotif, logger.With(logkeys.Handler, "put-declaration")),
				"PUT",
			)

			mux.Handle(
				"/v1/declarations/:id",
				apihttp.GetDeclarationHandler(store, logger.With(logkeys.Handler, "get-declaration")),
				"GET",
			)

			mux.Handle(
				"/v1/declarations/:id",
				apihttp.DeleteDeclarationHandler(store, logger.With(logkeys.Handler, "delete-declaration")),
				"DELETE",
			)

			mux.Handle(
				"/v1/declarations/:id/touch",
				apihttp.TouchDeclarationHandler(store, nanoNotif, logger.With(logkeys.Handler, "touch-declaration")),
				"POST",
			)

			// sets
			mux.Handle(
				"/v1/sets",
				apihttp.GetSetsHandler(store, logger.With("get-sets")),
				"GET",
			)

			// set declarations
			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.GetSetDeclarationsHandler(store, logger.With(logkeys.Handler, "get-set-declarations")),
				"GET",
			)

			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.PutSetDeclarationHandler(store, nanoNotif, logger.With(logkeys.Handler, "put-set-declarations")),
				"PUT",
			)

			mux.Handle(
				"/v1/set-declarations/:id",
				apihttp.DeleteSetDeclarationHandler(store, nanoNotif, logger.With(logkeys.Handler, "delete-set-delcarations")),
				"DELETE",
			)

			// enrollment sets
			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.GetEnrollmentSetsHandler(store, logger.With(logkeys.Handler, "get-enrollment-sets")),
				"GET",
			)

			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.PutEnrollmentSetHandler(store, nanoNotif, logger.With(logkeys.Handler, "put-enrollment-sets")),
				"PUT",
			)

			mux.Handle(
				"/v1/enrollment-sets/:id",
				apihttp.DeleteEnrollmentSetHandler(store, nanoNotif, logger.With(logkeys.Handler, "delete-enrollment-sets")),
				"DELETE",
			)

			// declarations sets
			mux.Handle(
				"/v1/declaration-sets/:id",
				apihttp.GetDeclarationSetsHandler(store, logger.With(logkeys.Handler, "get-declaration-sets")),
				"GET",
			)

			// status queries
			mux.Handle(
				"/v1/declaration-status/:id",
				apihttp.GetDeclarationStatusHandler(store, logger.With(logkeys.Handler, "get-declaration-status")),
				"GET",
			)

			mux.Handle(
				"/v1/status-errors/:id",
				apihttp.GetStatusErrorsHandler(store, logger.With(logkeys.Handler, "get-status-errors")),
				"GET",
			)

			mux.Handle(
				"/v1/status-values/:id",
				apihttp.GetStatusValuesHandler(store, logger.With(logkeys.Handler, "get-status-values")),
				"GET",
			)

			mux.Handle(
				"/v1/status-report/:id",
				apihttp.GetStatusReportHandler(store, logger.With(logkeys.Handler, "get-status-report")),
				"GET",
			)

			// notifier
			mux.Handle(
				"/v1/notify",
				apihttp.NotifyHandler(nanoNotif, logger.With(logkeys.Handler, "notify")),
				"POST",
			)
		})
	}

	// init for newTraceID()
	rand.Seed(time.Now().UnixNano())

	logger.Info(logkeys.Message, "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, httpddm.TraceLoggingMiddleware(mux, logger.With(logkeys.Handler, "log"), newTraceID))
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
