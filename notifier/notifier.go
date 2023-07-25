// Package notifier notifies enrollments of changed declarations.
package notifier

import (
	"context"

	"github.com/groob/plist"
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

type Enqueuer interface {
	// EnqueueDMCommand enqueues a DeclarativeManagement command to ids optionally using tokensJSON.
	EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error
	// SupportsMultiCommands() bool
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

	var tokensJSON []byte
	var tokens bool

	if len(ids) == 1 && n.sendTokens {
		tokensJSON, err = n.store.RetrieveTokensJSON(ctx, ids[0])
		if err != nil {
			return err
		}
		if len(tokensJSON) > 0 {
			tokens = true
		}
	}

	ctxlog.Logger(ctx, n.logger).Debug(
		logkeys.Message, "enqueueing command",
		logkeys.GenericCount, len(ids),
		logkeys.FirstEnrollmentID, ids[0],
		"tokens", tokens,
	)

	// TODO: consider checking enqueuer for SupportsMultiCommands and
	// sending individual EnqueueDMCommands (i.e.) n.sendTokens
	return n.enqueuer.EnqueueDMCommand(ctx, ids, tokensJSON)
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
