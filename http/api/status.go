package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// GetDeclarationStatusHandler returns a handler that retrives that last declaration status for an enrollment ID.
func GetDeclarationStatusHandler(store storage.StatusDeclarationsRetriever, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveDeclarationStatus(ctx, strings.Split(resource, ","))
		},
	)
}

// GetStatusErrorsHandler returns a handler that retrieves the collected errors for an enrollment.
func GetStatusErrorsHandler(store storage.StatusErrorsRetriever, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			if store == nil {
				return nil, errors.New("nil storage")
			}
			return store.RetrieveStatusErrors(ctx, strings.Split(resource, ","), 0, 10)
		},
	)
}

// GetStatusValuesHandler returns a handler that retrieves the collected values for an enrollment.
func GetStatusValuesHandler(store storage.StatusValuesRetriever, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL) (interface{}, error) {
			if store == nil {
				return nil, errors.New("nil storage")
			}
			pathPrefix := u.Query().Get("prefix")
			return store.RetrieveStatusValues(ctx, strings.Split(resource, ","), pathPrefix)
		},
	)
}

// GetStatusReportHandler returns a handler that retrieves a status report for en enrollment.
func GetStatusReportHandler(store storage.StatusReportRetriever, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		q := storage.StatusReportQuery{EnrollmentID: getResourceID(r)}
		if _, ok := r.URL.Query()["index"]; ok {
			index64, err := strconv.ParseInt(r.URL.Query().Get("index"), 10, 64)
			if err != nil {
				jsonErrorAndLog(w, http.StatusBadRequest, err, "parsing index", logger)
				return
			}
			index := int(index64)
			q.Index = &index
		}
		if statusID := r.URL.Query().Get("status_id"); statusID != "" {
			q.StatusID = &statusID
		}
		report, err := store.RetrieveStatusReport(r.Context(), q)
		statusCode := 0
		if err == nil && report == nil {
			err = errors.New("status report not found")
			statusCode = 404
		}
		if err != nil {
			jsonErrorAndLog(w, statusCode, err, "retrieving status report", logger)
			return
		}
		w.Header().Set("Content-type", jsonContentType)
		if !report.Timestamp.IsZero() {
			w.Header().Set("Last-Modified", report.Timestamp.UTC().Format(http.TimeFormat))
		}
		w.Header().Set("X-Status-Report-Index", strconv.Itoa(report.Index))
		if report.StatusID != "" {
			w.Header().Set("X-Status-Report-ID", report.StatusID)
		}
		w.Write(report.Raw)
	}
}
