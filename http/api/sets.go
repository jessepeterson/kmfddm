package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// GetDeclarationSetsHandler retrieves the list of sets for an declaration ID.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func GetDeclarationSetsHandler(store storage.DeclarationSetRetriever, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveDeclarationSets(ctx, resource)
		},
	)
}

// GetSetDeclarationsHandler retrieves the list of declarations in a set.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func GetSetDeclarationsHandler(store storage.SetDeclarationsRetriever, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveSetDeclarations(ctx, resource)
		},
	)
}

// GetSetEnrollmentsHandler retrieves the list of MDM enrollment IDs associated to a set.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
//
// Warning: the response is not paginated and may be very large for sets
// assigned to many enrollments.
func GetSetEnrollmentsHandler(store storage.EnrollmentIDRetriever, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			ids, err := store.RetrieveEnrollmentIDs(ctx, nil, []string{resource}, nil)
			if err == nil && ids == nil {
				// encode as an empty JSON array instead of null
				ids = []string{}
			}
			return ids, err
		},
	)
}

// PutSetDeclarationHandler associates declarations to a set.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func PutSetDeclarationHandler(store storage.SetDeclarationStorer, notifier Notifier, logger log.Logger) http.HandlerFunc {
	if store == nil || notifier == nil || logger == nil {
		panic("nil store or notifier or logger")
	}
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL, notify bool) (bool, string, error) {
			declarationID := u.Query().Get("declaration")
			if declarationID == "" {
				return false, "", errors.New("empty declaration")
			}
			changed, err := store.StoreSetDeclaration(ctx, resource, declarationID)
			if err == nil && changed && notify {
				err = notifier.Changed(ctx, nil, []string{resource}, nil)
				if err != nil {
					err = fmt.Errorf("notify set: %w", err)
				}
			}
			return changed, "store set declaration", err
		},
	)
}

// DeleteSetDeclarationHandler dissociates declarations from a set.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func DeleteSetDeclarationHandler(store storage.SetDeclarationRemover, notifier Notifier, logger log.Logger) http.HandlerFunc {
	if store == nil || notifier == nil || logger == nil {
		panic("nil store or notifier or logger")
	}
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL, notify bool) (bool, string, error) {
			declarationID := u.Query().Get("declaration")
			if declarationID == "" {
				return false, "", errors.New("empty declaration")
			}
			changed, err := store.RemoveSetDeclaration(ctx, resource, declarationID)
			if err == nil && changed && notify {
				err = notifier.Changed(ctx, nil, []string{resource}, nil)
				if err != nil {
					err = fmt.Errorf("notify set: %w", err)
				}
			}
			return changed, "remove set declaration", err
		},
	)
}

// GetSetsHandler returns a handler that retrieves the list of sets.
func GetSetsHandler(store storage.SetRetreiver, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		ids, err := store.RetrieveSets(r.Context())
		if err != nil {
			jsonErrorAndLog(w, 0, err, "retrieving sets", logger)
		}
		w.Header().Set("Content-type", jsonContentType)
		err = json.NewEncoder(w).Encode(&ids)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}
