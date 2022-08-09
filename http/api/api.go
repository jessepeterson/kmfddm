package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
)

const (
	jsonContentType = "application/json"
)

func jsonResponse(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-type", jsonContentType)
	if status > 0 {
		w.WriteHeader(status)
	}
	return json.NewEncoder(w).Encode(v)
}

type jsonErrorStruct struct {
	Err string `json:"error"`
}

func jsonError(w http.ResponseWriter, status int, err error) error {
	if status < 1 {
		status = http.StatusInternalServerError
	}
	return jsonResponse(w, status, &jsonErrorStruct{Err: err.Error()})
}

func jsonErrorAndLog(w http.ResponseWriter, status int, err error, msg string, logger log.Logger) {
	logger.Info("msg", msg, "err", err)
	err = jsonError(w, status, err)
	if err != nil {
		logger.Info("msg", "writing response json", "err", err)
	}
}

func simpleJSONResourceHandler(logger log.Logger, dataFn func(context.Context, string, *url.URL) (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		resource := getResourceID(r)
		if resource == "" {
			err = errors.New("empty resource ID")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "validating input", logger)
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

func simpleChangeResourceHandler(logger log.Logger, chgFn func(context.Context, string, *url.URL) (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		resource := getResourceID(r)
		if resource == "" {
			err = errors.New("empty resource ID")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "validating input", logger)
			return
		}
		logger = logger.With("resource", resource)
		if chgFn == nil {
			err = errors.New("no data change function defined")
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "validating input", logger)
			return
		}
		changed, err := chgFn(r.Context(), resource, r.URL)
		if err != nil {
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "retrieving data", logger)
			return
		}
		status := http.StatusNotModified
		if changed {
			status = http.StatusNoContent
		}
		// not actually an error, using as a helper
		http.Error(w, http.StatusText(status), status)
	}
}

func getResourceID(r *http.Request) string {
	return r.URL.Path
}

type Notifier interface {
	DeclarationChanged(ctx context.Context, identifier string) error
	EnrollmentChanged(ctx context.Context, enrollID string) error
	SetChanged(ctx context.Context, setName string) error
}
