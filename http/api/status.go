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

type StatusAPIStorage interface {
	RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]storage.StatusError, error)
	RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]storage.StatusValue, error)
}

// GetDeclarationStatusHandler returns a handler that retrives that last declaration status for an enrollment ID.
func GetDeclarationStatusHandler(store storage.StatusDeclarationsRetriever, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveDeclarationStatus(ctx, strings.Split(resource, ","))
		},
	)
}

func GetStatusErrorsHandler(store StatusAPIStorage, logger log.Logger) http.HandlerFunc {
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

func GetStatusValuesHandler(store StatusAPIStorage, logger log.Logger) http.HandlerFunc {
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
