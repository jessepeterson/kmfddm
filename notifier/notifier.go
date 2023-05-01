// Package notifier notifies devices of changed declarations or declaration
// items by sending them Declaration
package notifier

import (
	"context"
	"net/http"

	"github.com/jessepeterson/kmfddm/log"
)

// EnrollmentIDFinder is the interface we use to fetch enrollment IDs.
type EnrollmentIDFinder interface {
	RetrieveDeclarationEnrollmentIDs(ctx context.Context, declarationID string) ([]string, error)
	RetrieveSetEnrollmentIDs(ctx context.Context, setName string) ([]string, error)
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

type Notifier struct {
	store  EnrollmentIDFinder
	url    string
	user   string
	method string
	key    string
	logger log.Logger

	multi                 bool
	sendTokensForSingleID bool
}

type Option func(n *Notifier)

func WithMicroMDM() Option {
	return func(n *Notifier) {
		n.user = "micromdm"
		n.multi = false
		n.method = http.MethodPost
	}
}

func WithLogger(logger log.Logger) Option {
	return func(n *Notifier) {
		n.logger = logger
	}
}

func New(store EnrollmentIDFinder, url, key string, opts ...Option) *Notifier {
	n := &Notifier{
		store:                 store,
		url:                   url,
		key:                   key,
		logger:                log.NopLogger,
		sendTokensForSingleID: true,

		user:   "nanomdm",
		method: http.MethodPut,
		multi:  true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

func (n *Notifier) DeclarationChanged(ctx context.Context, declarationID string) error {
	ids, err := n.store.RetrieveDeclarationEnrollmentIDs(ctx, declarationID)
	if err != nil {
		return err
	}
	if len(ids) < 1 {
		n.logger.Debug("msg", "no enrollments to notify")
		return nil
	}
	return n.sendCommand(ctx, ids)
}

func (n *Notifier) EnrollmentChanged(ctx context.Context, enrollID string) error {
	return n.sendCommand(ctx, []string{enrollID})
}

func (n *Notifier) SetChanged(ctx context.Context, setName string) error {
	ids, err := n.store.RetrieveSetEnrollmentIDs(ctx, setName)
	if err != nil {
		return err
	}
	if len(ids) < 1 {
		n.logger.Debug("msg", "no enrollments to notify")
		return nil
	}
	return n.sendCommand(ctx, ids)
}
