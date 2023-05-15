package notifier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/groob/plist"
)

const agent = "kmfddm/0"

func (n *Notifier) sendCommand(ctx context.Context, ids []string) error {
	if len(ids) < 1 {
		return errors.New("sending command: no ids")
	}

	var cmdIDs [][]string
	if n.multi {
		// multiple command targeting support
		cmdIDs = [][]string{ids}
	} else {
		// split each id into a separate array
		for _, id := range ids {
			cmdIDs = append(cmdIDs, []string{id})
		}
	}

	for _, ids := range cmdIDs {
		if len(ids) < 1 {
			continue
		}
		logs := []interface{}{
			"msg", "sending command",
			"count", len(ids),
			"id_first", ids[0],
		}

		var err error
		var tokens []byte
		if len(ids) == 1 && n.sendTokensForSingleID {
			logs = append(logs, "include tokens", true)
			tokens, err = n.store.RetrieveTokensJSON(ctx, ids[0])
			if err != nil {
				errLogs := append(logs,
					"err",
					fmt.Errorf("retrieving tokens json: %w", err),
				)
				n.logger.Info(errLogs...)
				continue
			}
		}

		c := NewDeclarativeManagementCommand(uuid.NewString())
		if len(tokens) > 0 {
			c.Command.Data = &tokens
		}

		logs = append(logs, "command_uuid", c.CommandUUID)

		cmdBytes, err := plist.Marshal(c)
		if err != nil {
			errLogs := append(logs,
				"err",
				fmt.Errorf("marshal command plist: %w", err),
			)
			n.logger.Info(errLogs...)
			continue
		}

		var cmd io.Reader
		if len(cmdBytes) > 0 {
			cmd = bytes.NewBuffer(cmdBytes)
		}

		// TODO: this probably won't scale well. the URL could proabably
		// reasonably store 2K of data, so for a typical UUID this works out
		// to 50-ish IDs in a single enqueuing.
		urlFramgent, err := url.Parse(strings.Join(ids, ","))
		if err != nil {
			n.logger.Info("msg", "parsing url ids", "err", err)
			continue
		}
		url := n.url.ResolveReference(urlFramgent)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url.String(), cmd)
		if err != nil {
			errLogs := append(logs,
				"err",
				fmt.Errorf("new request: %w", err),
			)
			n.logger.Info(errLogs...)
			continue
		}
		req.SetBasicAuth(n.user, n.key)
		req.Header.Set("User-Agent", agent)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errLogs := append(logs,
				"err",
				fmt.Errorf("http reqeust: %w", err),
			)
			n.logger.Info(errLogs...)
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			n.logger.Info("msg", "closing body", "err", err)
		}

		logs = append(logs, "http_status", resp.StatusCode)

		// TODO: read the success or failure of the command enqueing/pushing and report/error on it.
		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			errLogs := append(logs,
				"err",
				fmt.Errorf("HTTP status: %s", resp.Status),
			)
			n.logger.Info(errLogs...)
			continue
		}

		n.logger.Debug(logs...)
	}

	return nil
}
