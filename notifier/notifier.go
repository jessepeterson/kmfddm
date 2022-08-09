// Package notifier notifies devices of changed declarations or declaration
// items by sending them Declaration
package notifier

import (
	"context"

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
	key    string
	logger log.Logger

	sendTokensForSingleID bool
}

func New(store EnrollmentIDFinder, url, key string, logger log.Logger) *Notifier {
	return &Notifier{
		store:                 store,
		url:                   url,
		key:                   key,
		logger:                logger,
		sendTokensForSingleID: true,
	}
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
