package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/storage"
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
