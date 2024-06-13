// Pacakge api contains HTTP handlers for working with KMFDDM data.
// This includes delcarations, sets, enrollments, status data,
// notifications, etc.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jessepeterson/kmfddm/logkeys"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

type Notifier interface {
	Changed(ctx context.Context, declarations []string, sets []string, ids []string) error
}

const (
	jsonContentType = "application/json"
)

var ErrEmptyResourceID = errors.New("empty resource id")

// jsonResponse encodes v to JSON and writes to w.
// If a non-zero HTTP status is provided it is written it to w.
func jsonResponse(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-type", jsonContentType)
	if status > 0 {
		w.WriteHeader(status)
	}
	return json.NewEncoder(w).Encode(v)
}

// jsonErrorStruct is encoded and output for HTTP errors.
type jsonErrorStruct struct {
	Err string `json:"error"`
}

// jsonError encodes err to JSON and writes to w.
// Status defaults to Internal Server Error if a positive HTTP status
// is not provided.
func jsonError(w http.ResponseWriter, status int, err error) error {
	if status < 1 {
		status = http.StatusInternalServerError
	}
	return jsonResponse(w, status, &jsonErrorStruct{Err: err.Error()})
}

// jsonErrorAndLog logs msg to logger then writes the JSON error to w.
func jsonErrorAndLog(w http.ResponseWriter, status int, err error, msg string, logger log.Logger) {
	logger.Info(logkeys.Message, msg, logkeys.Error, err)
	if err = jsonError(w, status, err); err != nil {
		logger.Info(logkeys.Message, "writing error json", logkeys.Message, err)
	}
}

func boolish(s string) bool {
	switch strings.ToLower(s) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

func shouldNotify(u *url.URL) bool {
	return !boolish(u.Query().Get("nonotify"))
}

func getResourceID(r *http.Request) string {
	return flow.Param(r.Context(), "id")
}

type dataFunc func(context.Context, string, *url.URL) (interface{}, error)

func simpleJSONResourceHandler(logger log.Logger, dataFn dataFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		resource := getResourceID(r)
		if resource == "" {
			jsonErrorAndLog(w, http.StatusBadRequest, ErrEmptyResourceID, "validating input", logger)
			return
		}
		logger = logger.With("resource", resource)
		if dataFn == nil {
			err = errors.New("no data retrieval function defined")
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "validating input", logger)
			return
		}
		data, err := dataFn(r.Context(), resource, r.URL)
		if err != nil {
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "retrieving data", logger)
			return
		}
		if data == nil {
			logger.Debug("msg", "no content")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-type", jsonContentType)
		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}

type changeFunc func(context.Context, string, *url.URL, bool) (bool, string, error)

func simpleChangeResourceHandler(logger log.Logger, chgFn changeFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		resource := getResourceID(r)
		if resource == "" {
			jsonErrorAndLog(w, http.StatusBadRequest, ErrEmptyResourceID, "validating input", logger)
			return
		}
		logger = logger.With("resource", resource)
		if chgFn == nil {
			err = errors.New("no data change function defined")
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "validating input", logger)
			return
		}
		notify := shouldNotify(r.URL)
		changed, dataName, err := chgFn(r.Context(), resource, r.URL, notify)
		chFnLogger := logger.With("msg", dataName, "changed", changed, "notify", changed && notify)
		if err != nil {
			chFnLogger.Info("err", err)
			err = jsonError(w, http.StatusInternalServerError, err)
			if err != nil {
				logger.Info("msg", "writing response json", "err", err)
			}
			return
		} else {
			chFnLogger.Debug()
		}
		status := http.StatusNotModified
		if changed {
			status = http.StatusNoContent
		}
		// not actually an error, using as a helper
		http.Error(w, http.StatusText(status), status)
	}
}
