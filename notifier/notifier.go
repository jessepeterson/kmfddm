// Package notifier notifies enrollments of changed declarations.
package notifier

import (
	"context"

	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/log/logkeys"
	"github.com/jessepeterson/kmfddm/storage"
)

// EnrollmentIDFinder is the interface we use to fetch enrollment IDs.
type EnrollmentIDFinder interface {
	storage.TokensJSONRetriever
	storage.EnrollmentIDRetriever
}

type Notifier struct {
	enqueuer Enqueuer
	store    EnrollmentIDFinder
	logger   log.Logger

	sendTokensForSingleID bool
}

type Option func(n *Notifier)

func WithLogger(logger log.Logger) Option {
	return func(n *Notifier) {
		n.logger = logger
	}
}

func New(enqueuer Enqueuer, store EnrollmentIDFinder, opts ...Option) (*Notifier, error) {
	n := &Notifier{
		enqueuer:              enqueuer,
		store:                 store,
		logger:                log.NopLogger,
		sendTokensForSingleID: true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

func (n *Notifier) Changed(ctx context.Context, declarations []string, sets []string, ids []string) error {
	idsOut, err := n.store.RetrieveEnrollmentIDs(ctx, declarations, sets, ids)
	if err != nil {
		return err
	}
	if len(idsOut) < 1 {
		ctxlog.Logger(ctx, n.logger).Debug(logkeys.Message, "no enrollments to notify")
		return nil
	}
	return n.sendCommand(ctx, idsOut)
}
