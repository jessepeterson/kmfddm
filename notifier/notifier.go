// Package notifier notifies enrollments of changed declarations.
package notifier

import (
	"context"
	"fmt"

	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
	"github.com/micromdm/plist"
)

// EnrollmentIDFinder is the interface we use to fetch enrollment IDs.
type EnrollmentIDFinder interface {
	storage.TokensJSONRetriever
	storage.EnrollmentIDRetriever
}

type Enqueuer interface {
	// EnqueueDMCommand enqueues a DeclarativeManagement command to ids optionally using tokensJSON.
	EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error

	// SupportsMultiCommands reports whether the enqueuer supports
	// multi-targeted commands.
	// These are commands that can be sent to multiple devices (i.e. using
	// the same enrollment ID).
	SupportsMultiCommands() bool
}

// Notifier enqueues DM commands to enrollments based on changes.
type Notifier struct {
	enqueuer   Enqueuer
	store      EnrollmentIDFinder
	logger     log.Logger
	sendTokens bool
}

type Option func(n *Notifier)

func WithLogger(logger log.Logger) Option {
	return func(n *Notifier) {
		n.logger = logger
	}
}

func New(enqueuer Enqueuer, store EnrollmentIDFinder, opts ...Option) (*Notifier, error) {
	if enqueuer == nil || store == nil {
		panic("enqueuer nor store can be nil")
	}
	n := &Notifier{
		enqueuer:   enqueuer,
		store:      store,
		logger:     log.NopLogger,
		sendTokens: true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

// Change notifies (enqueues the DM command to) enrollments for which the changes apply to.
func (n *Notifier) Changed(ctx context.Context, declarations []string, sets []string, idsIn []string) error {
	ids, err := n.store.RetrieveEnrollmentIDs(ctx, declarations, sets, idsIn)
	if err != nil {
		return err
	}
	if len(ids) < 1 {
		ctxlog.Logger(ctx, n.logger).Debug(logkeys.Message, "no enrollments to notify")
		return nil
	}

	ctxlog.Logger(ctx, n.logger).Debug(
		logkeys.Message, "enqueueing command",
		logkeys.GenericCount, len(ids),
		logkeys.FirstEnrollmentID, ids[0],
		"tokens", n.sendTokens,
	)

	enqueue := func(ids []string) error {
		var tokensJSON []byte
		var err error
		if len(ids) == 1 && n.sendTokens {
			tokensJSON, err = n.store.RetrieveTokensJSON(ctx, ids[0])
			if err != nil {
				return fmt.Errorf("retrieving tokens JSON: %w", err)
			}
		}

		if err = n.enqueuer.EnqueueDMCommand(ctx, ids, tokensJSON); err != nil {
			return fmt.Errorf("enqueueing DM command: %w", err)
		}

		return nil
	}

	if !n.enqueuer.SupportsMultiCommands() {
		// consider making this some kind of worker pool?
		for i, id := range ids {
			if err = enqueue([]string{id}); err != nil {
				return fmt.Errorf("enqueueing command %d/%d: %w", i+1, len(ids), err)
			}
		}

		return nil
	}

	return enqueue(ids)
}

// MakeCommand returns a raw MDM command in plist form using uuid and optionally tokensJSON.
func MakeCommand(uuid string, tokensJSON []byte) ([]byte, error) {
	c := NewDeclarativeManagementCommand(uuid)
	if len(tokensJSON) > 0 {
		// populating the tokens JSON within the MDM command saves
		// the device from having to request the DDM tokens endpoint
		// itself. it's a way to "front-load" the tokens retrieval.
		c.Command.Data = &tokensJSON
	}
	return plist.Marshal(c)
}
