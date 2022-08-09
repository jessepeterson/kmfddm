package main

import (
	"flag"
	"fmt"
	"hash"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/cespare/xxhash"
	httpddm "github.com/jessepeterson/kmfddm/http"
	apihttp "github.com/jessepeterson/kmfddm/http/api"
	ddmhttp "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/log/stdlogfmt"
	"github.com/jessepeterson/kmfddm/notifier"
	"github.com/jessepeterson/kmfddm/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	apiUsername = "kmfddm"
	apiRealm    = "kmfddm"

	endpointVersion = "/version"

	// DDM endpoints for devices/enrollments (passed from MDM)
	endpointDeclaration      = "/declaration/"
	endpointDeclarationItems = "/declaration-items"
	endpointTokens           = "/tokens"
	endpointStatus           = "/status"

	// API endpoints
	endpointDeclarations        = "/v1/declarations"
	endpointDeclarationsID      = "/v1/declarations/"
	endpointSets                = "/v1/sets"
	endpointSetDeclarationsID   = "/v1/set-declarations/"
	endpointEnrollmentSetsID    = "/v1/enrollment-sets/"
	endpointDeclarationSetsID   = "/v1/declaration-sets/"
	endpointDeclarationStatusID = "/v1/declaration-status/"
	endpointStatusErrorsID      = "/v1/status-errors/"
	endpointStatusValuesID      = "/v1/status-values/"
)

func main() {
	var (
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flListen  = flag.String("listen", ":9002", "HTTP listen address")
		flAPIKey  = flag.String("api", "", "API key for API endpoints")
		flVersion = flag.Bool("version", false, "print version")
		// flStorage = flag.String("storage", "", "storage backend")
		flDSN = flag.String("storage-dsn", "", "storage data source name")

		flDumpStatus = flag.Bool("dump-status", false, "dump the status report to stdout")

		flEnqueueURL = flag.String("enqueue", "http://[::1]:9000/v1/enqueue/", "URL of NanoMDM enqueue endpoint")
		flEnqueueKey = flag.String("enqueue-key", "", "NanoMDM API key")
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

	storage, err := mysql.New(
		mysql.WithDSN(*flDSN),
		mysql.WithNewHash(func() hash.Hash { return xxhash.New() }),
	)
	if err != nil {
		logger.Info("msg", "init MySQL storage", "err", err)
		os.Exit(1)
	}

	nanoNotif := notifier.New(storage, *flEnqueueURL, *flEnqueueKey, logger.With("service", "notifier"))

	mux := http.NewServeMux()

	mux.Handle(endpointVersion, httpddm.VersionHandler(version))

	var diHandler http.Handler
	diHandler = ddmhttp.TokensDeclarationItemsHandler(storage, false, logger.With("handler", "declaration-items"))
	mux.Handle(endpointDeclarationItems, diHandler)

	var tokHandler http.Handler
	tokHandler = ddmhttp.TokensDeclarationItemsHandler(storage, true, logger.With("handler", "tokens"))
	mux.Handle(endpointTokens, tokHandler)

	var sHandler http.Handler
	sHandler = ddmhttp.StatusReportHandler(storage, logger.With("handler", "status"))
	if *flDumpStatus {
		sHandler = DumpHandler(sHandler, os.Stdout)
	}
	mux.Handle(endpointStatus, sHandler)

	var dHandler http.Handler
	dHandler = ddmhttp.DeclarationHandler(storage, logger.With("handler", "declaration"))
	dHandler = http.StripPrefix(endpointDeclaration, dHandler)
	mux.Handle(endpointDeclaration, dHandler)

	if *flAPIKey != "" {
		declsMux := httpddm.NewMethodMux()
		declsMux.Handle("PUT", apihttp.PutDeclarationHandler(storage, nanoNotif, logger.With("handler", "put-declaration")))
		declsMux.Handle("GET", apihttp.GetDeclarationsHandler(storage, logger.With("get-declarations")))
		declsHandler := httpddm.BasicAuthMiddleware(declsMux, apiUsername, *flAPIKey, apiRealm)
		mux.Handle(endpointDeclarations, declsHandler)

		var setsHandler http.Handler
		setsHandler = apihttp.GetSetsHandler(storage, logger.With("get-sets"))
		setsHandler = httpddm.BasicAuthMiddleware(setsHandler, apiUsername, *flAPIKey, apiRealm)
		mux.Handle(endpointSets, setsHandler)

		handleStrippedAPI := func(prefix string, h http.Handler) {
			h = http.StripPrefix(prefix, h)
			h = httpddm.BasicAuthMiddleware(h, apiUsername, *flAPIKey, apiRealm)
			mux.Handle(prefix, h)
		}

		declMux := httpddm.NewMethodMux()
		declMux.Handle("GET", apihttp.GetDeclarationHandler(storage, logger.With("handler", "get-declaration")))
		declMux.Handle("DELETE", apihttp.DeleteDeclarationHandler(storage, logger.With("handler", "delete-declaration")))
		handleStrippedAPI(endpointDeclarationsID, declMux)

		setDeclMux := httpddm.NewMethodMux()
		setDeclMux.Handle("GET", apihttp.GetSetDeclarationsHandler(storage, logger.With("handler", "get-set-declarations")))
		setDeclMux.Handle("PUT", apihttp.PutSetDeclarationHandler(storage, nanoNotif, logger.With("handler", "put-set-declarations")))
		setDeclMux.Handle("DELETE", apihttp.DeleteSetDeclarationHandler(storage, nanoNotif, logger.With("handler", "delete-set-delcarations")))
		handleStrippedAPI(endpointSetDeclarationsID, setDeclMux)

		enrSetMux := httpddm.NewMethodMux()
		enrSetMux.Handle("GET", apihttp.GetEnrollmentSetsHandler(storage, logger.With("handler", "get-enrollment-sets")))
		enrSetMux.Handle("PUT", apihttp.PutEnrollmentSetHandler(storage, nanoNotif, logger.With("handler", "put-enrollment-sets")))
		enrSetMux.Handle("DELETE", apihttp.DeleteEnrollmentSetHandler(storage, nanoNotif, logger.With("handler", "delete-enrollment-sets")))
		handleStrippedAPI(endpointEnrollmentSetsID, enrSetMux)

		var dsHandler http.Handler
		dsHandler = apihttp.GetDeclarationSetsHandler(storage, logger.With("handler", "get-declaration-sets"))
		handleStrippedAPI(endpointDeclarationSetsID, dsHandler)

		var siHandler http.Handler
		siHandler = apihttp.GetDeclarationStatusHandler(storage, logger.With("handler", "get-declaration-status"))
		handleStrippedAPI(endpointDeclarationStatusID, siHandler)

		var seHandler http.Handler
		seHandler = apihttp.GetStatusErrorsHandler(storage, logger.With("handler", "get-status-errors"))
		handleStrippedAPI(endpointStatusErrorsID, seHandler)

		var svHandler http.Handler
		svHandler = apihttp.GetStatusValuesHandler(storage, logger.With("handler", "get-status-values"))
		handleStrippedAPI(endpointStatusValuesID, svHandler)
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
