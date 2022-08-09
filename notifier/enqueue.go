package notifier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const agent = "kmfddm/0"

func (n *Notifier) sendCommand(ctx context.Context, ids []string) error {
	if len(ids) < 1 {
		return errors.New("sending command: no ids")
	}

	logs := []interface{}{"msg", "sending command", "count", len(ids)}
	defer func() { n.logger.Info(logs...) }()

	var err error
	var tokens []byte
	if n.sendTokensForSingleID && len(ids) == 1 {
		logs = append(logs, "include tokens", true)
		tokens, err = n.store.RetrieveTokensJSON(ctx, ids[0])
		if err != nil {
			logs = append(logs, "err", err)
			return fmt.Errorf("retrieving tokens json: %w", err)
		}
	}

	cmdUUID := uuid.NewString()
	logs = append(logs, "command_uuid", cmdUUID)
	cmdBytes, err := NewMDMCommand(cmdUUID, tokens)
	if err != nil {
		logs = append(logs, "err", err)
		return err
	}

	var cmd io.Reader
	if len(cmdBytes) > 0 {
		cmd = bytes.NewBuffer(cmdBytes)
	}

	// TODO: this probably won't scale well. the URL could proabably
	// reasonably store 2K of data, so for a typical UUID this works out
	// to 50-ish IDs in a single enqueuing.
	idsStr := strings.Join(ids, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, n.url+idsStr, cmd)
	if err != nil {
		logs = append(logs, "err", err)
		return err
	}
	req.SetBasicAuth("nanomdm", n.key)
	req.Header.Set("User-Agent", agent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logs = append(logs, "err", err)
		return err
	}
	defer resp.Body.Close()
	logs = append(logs, "http_status", resp.StatusCode)

	// TODO: read the success or failure of the command enqueing/pushing and report/error on it.
	if resp.StatusCode != 200 {
		logs = append(logs, "err", err)
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}

	return nil
}
