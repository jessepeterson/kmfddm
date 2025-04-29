package e2e

import (
	"context"

	"github.com/jessepeterson/kmfddm/storage"
)

type captureNotifier struct {
	store   storage.EnrollmentIDRetriever
	lastIDs []string
	called  bool
}

func (n *captureNotifier) Changed(ctx context.Context, declarations []string, sets []string, ids []string) (err error) {
	n.called = true
	n.lastIDs, err = n.store.RetrieveEnrollmentIDs(ctx, declarations, sets, ids)
	return
}

func (n *captureNotifier) getAndClear() []string {
	r := n.lastIDs
	n.lastIDs = nil
	n.called = false
	return r
}
