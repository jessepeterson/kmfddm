// Package notifier notifies enrollments of changed declarations.
package notifier

import (
	"context"
	"net/http"
	"net/url"
	"strings"

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

type Notifier struct {
	store  EnrollmentIDFinder
	url    *url.URL
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

func New(store EnrollmentIDFinder, urlBase, key string, opts ...Option) (*Notifier, error) {
	n := &Notifier{
		store:                 store,
		key:                   key,
		logger:                log.NopLogger,
		sendTokensForSingleID: true,

		user:   "nanomdm",
		method: http.MethodPut,
		multi:  true,
	}
	var err error
	if !strings.HasSuffix(urlBase, "/") {
		urlBase += "/"
	}
	n.url, err = url.Parse(urlBase)
	if err != nil {
		return n, err
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

func (n *Notifier) Changed(ctx context.Context, declarations []string, sets []string, ids []string) error {
	idsOut, err := n.store.RetrieveEnrollmentIDs(ctx, declarations, sets, ids)
	if err != nil {
		return err
	}
	if len(idsOut) < 1 {
		ctxlog.Logger(ctx, n.logger).Debug(logkeys.Message, "no enrollments to notify")
		return nil
	}
	return n.sendCommand(ctx, idsOut)
}
