package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/storage"
)

// GetEnrollmentSetsHandler returns a handle that retrieves the list of sets for an enrollment ID.
func GetEnrollmentSetsHandler(store storage.EnrollmentSetsRetriever, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveEnrollmentSets(ctx, resource)
		},
	)
}

// PutEnrollmentSetHandler returns a handle that associates a set to an enrollment.
func PutEnrollmentSetHandler(store storage.EnrollmentSetStorer, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL, notify bool) (bool, string, error) {
			setName := u.Query().Get("set")
			if setName == "" {
				return false, "", errors.New("empty set name")
			}
			changed, err := store.StoreEnrollmentSet(ctx, resource, setName)
			if err == nil && changed && notify {
				err = notifier.Changed(ctx, nil, nil, []string{resource})
				if err != nil {
					err = fmt.Errorf("notify enrollment: %w", err)
				}
			}
			return changed, "store enrollment set", err
		},
	)
}

// DeleteEnrollmentSetHandler returns a handle that dissociates a set from an enrollment.
func DeleteEnrollmentSetHandler(store storage.EnrollmentSetRemover, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL, notify bool) (bool, string, error) {
			setName := u.Query().Get("set")
			if setName == "" {
				return false, "", errors.New("empty set name")
			}
			changed, err := store.RemoveEnrollmentSet(ctx, resource, setName)
			if err == nil && changed && notify {
				err = notifier.Changed(ctx, nil, nil, []string{resource})
				if err != nil {
					err = fmt.Errorf("notify enrollment: %w", err)
				}
			}
			return changed, "remove enrollment set", err
		},
	)
}
