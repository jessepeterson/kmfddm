package foss

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jessepeterson/kmfddm/notifier"
)

// EnqueueDMCommand enqueues a DeclarativeManagment command on the MDM server.
func (m *FossMDM) EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error {
	cmdBytes, err := notifier.MakeCommand(uuid.NewString(), tokensJSON)
	if err != nil {
		return fmt.Errorf("making command: %w", err)
	}
	return m.Enqueue(ctx, ids, cmdBytes)
}
