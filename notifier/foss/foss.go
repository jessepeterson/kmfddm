// Package foss implements communication with with Free/Open Source MDM servers.
package foss

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/log/logkeys"
)

var ErrNoIDsInIDChunk = errors.New("no ids in id chunk")

// Doer executes an HTTP request.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// FossMDM sends requests to Free/Open Source MDM servers for enqueueing MDM commands and sending APNs pushes.
// Ostensibly this means NanoMDM and MicroMDM servers, but any server
// that supports compatible API endpoints could work, too.
type FossMDM struct {
	logger log.Logger
	client Doer

	// maximum number of multi-targeted pushes or enqueueings supported.
	// if set to one this effectively disables multi-command enqueueings.
	max int

	user      string // HTTP Basic username
	apiKey    string // HTTP Basic password
	enqMethod string // HTTP method

	enqURL  *url.URL // "base" URL for enqueueing commands
	pushURL *url.URL // "base" URL for sending APNs pushes
}

type Option func(*FossMDM) error

func WithLogger(logger log.Logger) Option {
	return func(m *FossMDM) error {
		m.logger = logger
		return nil
	}
}

// WithMicroMDM uses MicroMDM API conventions.
func WithMicroMDM() Option {
	return func(m *FossMDM) error {
		m.max = 1
		m.user = "micromdm"
		m.enqMethod = http.MethodPost
		return nil
	}
}

func prepURL(ref string) (*url.URL, error) {
	if !strings.HasSuffix(ref, "/") {
		// endpoints work by concatenating enrollment ID(s) to the base
		// URLs as an additional "path." Make sure our URLs end with /
		// to make this work properly.
		ref += "/"
	}
	return url.Parse(ref)
}

// WithPush configures sending APNs push requests to ref base URL.
func WithPush(ref string) Option {
	return func(m *FossMDM) (err error) {
		m.pushURL, err = prepURL(ref)
		if err != nil {
			err = fmt.Errorf("preparing push URL: %w", err)
		}
		return
	}
}

// 30 is a conservative estimate for a reasonable number of URL
// parameters in a request path considering typical limitations.
const defaultMaxIDs = 30

// NewFossMDM creates a new FossMDM. The enqueue and push URL "base" is
// specified with enqRef. By default we target NanoMDM conventions.
func NewFossMDM(enqRef, apiKey string, opts ...Option) (*FossMDM, error) {
	m := &FossMDM{
		client: http.DefaultClient,
		logger: log.NopLogger,

		max: defaultMaxIDs,

		user:      "nanomdm",
		apiKey:    apiKey,
		enqMethod: http.MethodPut,
	}
	var err error
	m.enqURL, err = prepURL(enqRef)
	if err != nil {
		return m, fmt.Errorf("preparing enqueue URL: %w", err)
	}
	for _, opt := range opts {
		err = opt(m)
		if err != nil {
			return m, fmt.Errorf("processing option: %w", err)
		}
	}
	return m, nil
}

// SupportsMultiCommands reports whether we support multi-targeted commands.
// These are commands that can be sent to multiple devices (i.e. using
// the same UUID).
func (m *FossMDM) SupportsMultiCommands() bool {
	return m.max > 1
}

func concatURL(base *url.URL, ids []string) (string, error) {
	if base == nil {
		return "", errors.New("invalid base URL")
	}
	joinedIDs, err := url.Parse(strings.Join(ids, ","))
	if err != nil {
		return "", err
	}
	return base.ResolveReference(joinedIDs).String(), nil
}

func chunk(s []string, n int) (chunks [][]string) {
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return
}

// Enqueue sends the HTTP request to enqueue rawCommand to ids on the MDM server.
func (m *FossMDM) Enqueue(ctx context.Context, ids []string, rawCommand []byte) error {
	if m.max == 1 && len(ids) > 1 {
		// err on the side of caution so that we don't try to enqueue
		// the same command UUID onto different ids.
		return errors.New("multiple ids not supported")
	}
	buf := bytes.NewBuffer(rawCommand)
	logger := ctxlog.Logger(ctx, m.logger).With("request", "enqueue")
	// TODO: perhaps parallelize?
	for _, idChunk := range chunk(ids, m.max) {
		if len(idChunk) < 1 {
			logger.Info(logkeys.Error, ErrNoIDsInIDChunk)
			continue
		}
		idsLogger := logger.With(
			logkeys.GenericCount, len(idChunk),
			logkeys.FirstEnrollmentID, idChunk[0],
		)
		ref, err := concatURL(m.enqURL, idChunk)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "creating enqueue URL",
				logkeys.Error, err,
			)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, m.enqMethod, ref, buf)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "creating HTTP request",
				logkeys.Error, err,
			)
			continue
		}
		req.SetBasicAuth(m.user, m.apiKey)
		resp, err := m.client.Do(req)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "executing HTTP request",
				logkeys.Error, err,
			)
			continue
		}
		idsLogger.Debug(
			logkeys.Message, "enqueue command",
			"http_status_code", resp.StatusCode,
			"http_status", resp.Status,
		)
		if err = resp.Body.Close(); err != nil {
			idsLogger.Info(
				logkeys.Message, "closing body",
				logkeys.Error, err,
			)
		}
	}
	return nil
}

// Push sends the HTTP request to send APNs pushes to ids on the MDM server.
func (m *FossMDM) Push(ctx context.Context, ids []string) error {
	if m.pushURL == nil {
		return errors.New("push not configured")
	}
	logger := ctxlog.Logger(ctx, m.logger).With("request", "push")
	// TODO: perhaps parallelize?
	for _, idChunk := range chunk(ids, m.max) {
		if len(idChunk) < 1 {
			logger.Info(logkeys.Error, ErrNoIDsInIDChunk)
			continue
		}
		idsLogger := logger.With(
			logkeys.GenericCount, len(idChunk),
			logkeys.FirstEnrollmentID, idChunk[0],
		)
		ref, err := concatURL(m.pushURL, idChunk)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "creating push URL",
				logkeys.Error, err,
			)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "creating HTTP request",
				logkeys.Error, err,
			)
			continue
		}
		req.SetBasicAuth(m.user, m.apiKey)
		resp, err := m.client.Do(req)
		if err != nil {
			idsLogger.Info(
				logkeys.Message, "executing HTTP request",
				logkeys.Error, err,
			)
			continue
		}
		idsLogger.Debug(
			logkeys.Message, "push",
			"http_status_code", resp.StatusCode,
			"http_status", resp.Status,
		)
		if err = resp.Body.Close(); err != nil {
			idsLogger.Info(
				logkeys.Message, "closing body",
				logkeys.Error, err,
			)
		}
	}
	return nil
}
