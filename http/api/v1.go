package api

import (
	"net/http"

	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/micromdm/nanolib/log"
)

// Mux can register HTTP handlers.
// Ostensibly this supports flow router.
type Mux interface {
	// Handle registers the handler for the given pattern.
	Handle(pattern string, handler http.Handler, methods ...string)
}

// APIStorage is required for the API handlers.
type APIStorage interface {
	storage.DeclarationAPIStorage
	storage.SetDeclarationStorage
	storage.SetRetreiver
	storage.StatusAPIStorage
	storage.EnrollmentSetStorage
}

// func handlerName(endpoint string) string {
// 	return strings.Trim(endpoint, "/")
// }

// HandleAPIv1 registers the various API handlers into mux.
// API endpoint paths are prepended with prefix.
// Authentication or any other layered handlers are not present.
// They are assumed to be layered with mux, possibly at the Handel call.
// If prefix is empty and these handlers are used in sub-paths then
// handlers should have that sub-path stripped from the request.
// The logger is adorned with a "handler" key of the endpoint name.
func HandleAPIv1(prefix string, mux Mux, logger log.Logger, store APIStorage, notifier Notifier) {
	// declarations
	mux.Handle(
		prefix+"/declarations",
		GetDeclarationsHandler(store, logger.With("get-declarations")),
		"GET",
	)

	mux.Handle(
		prefix+"/declarations",
		PutDeclarationHandler(store, notifier, logger.With(logkeys.Handler, "put-declaration")),
		"PUT",
	)

	mux.Handle(
		prefix+"/declarations/:id",
		GetDeclarationHandler(store, logger.With(logkeys.Handler, "get-declaration")),
		"GET",
	)

	mux.Handle(
		prefix+"/declarations/:id",
		DeleteDeclarationHandler(store, logger.With(logkeys.Handler, "delete-declaration")),
		"DELETE",
	)

	mux.Handle(
		prefix+"/declarations/:id/touch",
		TouchDeclarationHandler(store, notifier, logger.With(logkeys.Handler, "touch-declaration")),
		"POST",
	)

	// sets
	mux.Handle(
		prefix+"/sets",
		GetSetsHandler(store, logger.With("get-sets")),
		"GET",
	)

	// set declarations
	mux.Handle(
		prefix+"/set-declarations/:id",
		GetSetDeclarationsHandler(store, logger.With(logkeys.Handler, "get-set-declarations")),
		"GET",
	)

	mux.Handle(
		prefix+"/set-declarations/:id",
		PutSetDeclarationHandler(store, notifier, logger.With(logkeys.Handler, "put-set-declarations")),
		"PUT",
	)

	mux.Handle(
		prefix+"/set-declarations/:id",
		DeleteSetDeclarationHandler(store, notifier, logger.With(logkeys.Handler, "delete-set-delcarations")),
		"DELETE",
	)

	// enrollment sets
	mux.Handle(
		prefix+"/enrollment-sets/:id",
		GetEnrollmentSetsHandler(store, logger.With(logkeys.Handler, "get-enrollment-sets")),
		"GET",
	)

	mux.Handle(
		prefix+"/enrollment-sets/:id",
		PutEnrollmentSetHandler(store, notifier, logger.With(logkeys.Handler, "put-enrollment-sets")),
		"PUT",
	)

	mux.Handle(
		prefix+"/enrollment-sets/:id",
		DeleteEnrollmentSetHandler(store, notifier, logger.With(logkeys.Handler, "delete-enrollment-sets")),
		"DELETE",
	)

	mux.Handle(
		prefix+"/enrollment-sets-all/sets/:id",
		DeleteAllEnrollmentSetsHandler(store, notifier, logger.With(logkeys.Handler, "delete-all-enrollment-sets")),
		"DELETE",
	)

	// declarations sets
	mux.Handle(
		prefix+"/declaration-sets/:id",
		GetDeclarationSetsHandler(store, logger.With(logkeys.Handler, "get-declaration-sets")),
		"GET",
	)

	// status queries
	mux.Handle(
		prefix+"/declaration-status/:id",
		GetDeclarationStatusHandler(store, logger.With(logkeys.Handler, "get-declaration-status")),
		"GET",
	)

	mux.Handle(
		prefix+"/status-errors/:id",
		GetStatusErrorsHandler(store, logger.With(logkeys.Handler, "get-status-errors")),
		"GET",
	)

	mux.Handle(
		prefix+"/status-values/:id",
		GetStatusValuesHandler(store, logger.With(logkeys.Handler, "get-status-values")),
		"GET",
	)

	mux.Handle(
		prefix+"/status-report/:id",
		GetStatusReportHandler(store, logger.With(logkeys.Handler, "get-status-report")),
		"GET",
	)

	// notifier
	mux.Handle(
		prefix+"/notify",
		NotifyHandler(notifier, logger.With(logkeys.Handler, "notify")),
		"POST",
	)
}
