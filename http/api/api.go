package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexedwards/flow"
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

func simpleChangeResourceHandler(logger log.Logger, chgFn func(context.Context, string, *url.URL, bool) (bool, string, error)) http.HandlerFunc {
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

func getResourceID(r *http.Request) string {
	return flow.Param(r.Context(), "id")
}

type Notifier interface {
	Changed(ctx context.Context, declarations []string, sets []string, ids []string) error
}
