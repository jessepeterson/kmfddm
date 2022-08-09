package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jessepeterson/kmfddm/log"
)

type EnrollmentAPIStorage interface {
	RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) (setNames []string, err error)
	StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)
	RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)
}

// GetEnrollmentSetsHandler retrieves the list of sets for an enrollment ID.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func GetEnrollmentSetsHandler(store EnrollmentAPIStorage, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveEnrollmentSets(ctx, resource)
		},
	)
}

// PutEnrollmentSetHandler associates a set to an enrollment.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func PutEnrollmentSetHandler(store EnrollmentAPIStorage, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL) (bool, error) {
			setName := u.Query().Get("set")
			if setName == "" {
				return false, errors.New("empty set name")
			}
			changed, err := store.StoreEnrollmentSet(ctx, resource, setName)
			if err == nil && changed {
				err = notifier.EnrollmentChanged(ctx, resource)
				if err != nil {
					err = fmt.Errorf("notify enrollment: %w", err)
				}
			}
			return changed, err
		},
	)
}

// DeleteEnrollmentSetHandler dissociates a set from an enrollment.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func DeleteEnrollmentSetHandler(store EnrollmentAPIStorage, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return simpleChangeResourceHandler(
		logger,
		func(ctx context.Context, resource string, u *url.URL) (bool, error) {
			setName := u.Query().Get("set")
			if setName == "" {
				return false, errors.New("empty set name")
			}
			changed, err := store.RemoveEnrollmentSet(ctx, resource, setName)
			if err == nil && changed {
				err = notifier.EnrollmentChanged(ctx, resource)
			}
			return changed, err
		},
	)
}
