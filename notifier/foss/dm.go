package foss

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jessepeterson/kmfddm/notifier"
)

// Enqueue sends the HTTP request to enqueue rawCommand to ids on the MDM server.
func (m *FossMDM) EnqueueCommand(ctx context.Context, ids []string, tokensJSON []byte) error {
	cmdBytes, err := notifier.MakeCommand(uuid.NewString(), tokensJSON)
	if err != nil {
		return fmt.Errorf("making command: %w", err)
	}
	return m.Enqueue(ctx, ids, cmdBytes)
}
