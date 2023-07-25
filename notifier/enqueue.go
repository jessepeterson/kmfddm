package notifier

import (
	"context"
	"errors"

	"github.com/groob/plist"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/log/logkeys"
)

type Enqueuer interface {
	// EnqueueDMCommand enqueues a DeclarativeManagement command to ids optionally using tokensJSON.
	EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error
}

// MakeCommand returns a raw MDM command in plist form using uuid and optionally tokensJSON.
func MakeCommand(uuid string, tokensJSON []byte) ([]byte, error) {
	c := NewDeclarativeManagementCommand(uuid)
	if len(tokensJSON) > 0 {
		c.Command.Data = &tokensJSON
	}
	return plist.Marshal(c)
}

func (n *Notifier) sendCommand(ctx context.Context, ids []string) error {
	if len(ids) < 1 {
		return errors.New("sending command: no ids")
	}

	var err error
	var tokensJSON []byte
	var tokens bool

	if len(ids) == 1 && n.sendTokensForSingleID {
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

	return n.enqueuer.EnqueueDMCommand(ctx, ids, tokensJSON)
}
