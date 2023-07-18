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
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/log/logkeys"
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

	logger := ctxlog.Logger(ctx, n.logger)

	for _, ids := range cmdIDs {
		if len(ids) < 1 {
			continue
		}
		logs := []interface{}{
			logkeys.Message, "sending command",
			logkeys.GenericCount, len(ids),
			logkeys.FirstEnrollmentID, ids[0],
		}

		var err error
		var tokens []byte
		if len(ids) == 1 && n.sendTokensForSingleID {
			logs = append(logs, "include tokens", true)
			tokens, err = n.store.RetrieveTokensJSON(ctx, ids[0])
			if err != nil {
				errLogs := append(logs,
					logkeys.Error,
					fmt.Errorf("retrieving tokens json: %w", err),
				)
				logger.Info(errLogs...)
				continue
			}
		}

		c := NewDeclarativeManagementCommand(uuid.NewString())
		if len(tokens) > 0 {
			c.Command.Data = &tokens
		}

		logs = append(logs, logkeys.CommandUUID, c.CommandUUID)

		cmdBytes, err := plist.Marshal(c)
		if err != nil {
			errLogs := append(logs,
				logkeys.Error,
				fmt.Errorf("marshal command plist: %w", err),
			)
			logger.Info(errLogs...)
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
			logger.Info(logkeys.Message, "parsing url ids", logkeys.Error, err)
			continue
		}
		url := n.url.ResolveReference(urlFramgent)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url.String(), cmd)
		if err != nil {
			errLogs := append(logs,
				logkeys.Error,
				fmt.Errorf("new request: %w", err),
			)
			logger.Info(errLogs...)
			continue
		}
		req.SetBasicAuth(n.user, n.key)
		req.Header.Set("User-Agent", agent)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errLogs := append(logs,
				logkeys.Error,
				fmt.Errorf("http reqeust: %w", err),
			)
			logger.Info(errLogs...)
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			logger.Info(logkeys.Message, "closing body", logkeys.Error, err)
		}

		logs = append(logs, "http_status", resp.StatusCode)

		// TODO: read the success or failure of the command enqueing/pushing and report/error on it.
		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			errLogs := append(logs,
				logkeys.Error,
				fmt.Errorf("HTTP status: %s", resp.Status),
			)
			logger.Info(errLogs...)
			continue
		}

		logger.Debug(logs...)
	}

	return nil
}
