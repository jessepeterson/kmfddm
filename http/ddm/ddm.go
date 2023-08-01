// Package ddm contains HTTP handlers for the Apple DDM protocol.
package ddm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/log/logkeys"
	"github.com/jessepeterson/kmfddm/storage"
)

const (
	// EnrollmentIDHeader is the enrollment ID HTTP header handed to us.
	EnrollmentIDHeader = "X-Enrollment-ID"

	jsonContentType = "application/json"
)

var ErrEmptyEnrollmentID = errors.New("empty enrollment ID")

// parseDeclarationPath parses path to separate out the declaration type and identifier.
func parseDeclarationPath(path string) (string, string, error) {
	split := strings.SplitN(path, "/", 2)
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid path element count: %d", len(split))
	}
	if split[0] == "" || split[1] == "" {
		return "", "", errors.New("empty type or identifier path elements")
	}
	return split[0], split[1], nil
}

func ErrorAndLog(w http.ResponseWriter, status int, logger log.Logger, msg string, err error) {
	logger.Info(logkeys.Message, msg, logkeys.Error, err)
	http.Error(w, http.StatusText(status), status)
}

type enrollmentIDContextKey struct{}

// contextEnrollmentID extracts the enrollment ID and sets up the context.
func contextEnrollmentID(r *http.Request, logger log.Logger) (context.Context, log.Logger, string, error) {
	id := r.Header.Get(EnrollmentIDHeader)
	if id == "" {
		return r.Context(), logger, id, ErrEmptyEnrollmentID
	}
	// add it to our context
	ctx := context.WithValue(r.Context(), enrollmentIDContextKey{}, id)
	// setup a new context logger KV func to
	ctx = ctxlog.AddFunc(ctx, ctxlog.SimpleStringFunc(logkeys.EnrollmentID, enrollmentIDContextKey{}))
	// return the new context and a context logger using it along with the id
	return ctx, ctxlog.Logger(ctx, logger), id, nil
}

// DeclarationHandler creates a handler that fetches and returns a single declaration.
// The request URL path is assumed to contain the declaration type and identifier.
// This probably requires the handler to have the path prefix stripped before use.
func DeclarationHandler(store storage.DeclarationRetriever, hLogger log.Logger) http.HandlerFunc {
	if store == nil || hLogger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger, enrollmentID, err := contextEnrollmentID(r, hLogger)
		if err != nil {
			ErrorAndLog(w, http.StatusBadRequest, logger, "getting enrollment id", err)
		}
		declarationType, declarationID, err := parseDeclarationPath(r.URL.Path)
		if err != nil {
			ErrorAndLog(w, http.StatusBadRequest, logger, "parsing path", err)
			return
		}
		logger = logger.With(
			logkeys.DeclarationID, declarationID,
			logkeys.DeclarationType, declarationType,
		)
		rawDecl, err := store.RetrieveEnrollmentDeclarationJSON(ctx, declarationID, declarationType, enrollmentID)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "retrieving declaration", err)
			return
		}
		logger.Debug(logkeys.Message, "retrieved declaration")
		w.Header().Set("Content-Type", jsonContentType)
		w.Write(rawDecl)
	}
}

// TokensOrDeclarationItemsHandler creates a handler that fetchs and returns either
// the tokens or declaration items JSON for an erollment ID depending on tokens.
func TokensOrDeclarationItemsHandler(store storage.TokensDeclarationItemsRetriever, tokens bool, hLogger log.Logger) http.HandlerFunc {
	if store == nil || hLogger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger, enrollmentID, err := contextEnrollmentID(r, hLogger)
		if err != nil {
			ErrorAndLog(w, http.StatusBadRequest, logger, "getting enrollment id", err)
		}
		var op string
		var rawJSON []byte
		if tokens {
			op = "tokens"
			rawJSON, err = store.RetrieveTokensJSON(ctx, enrollmentID)
		} else {
			op = "declaration items"
			rawJSON, err = store.RetrieveDeclarationItemsJSON(ctx, enrollmentID)
		}
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "retrieving "+op, err)
			return
		}
		logger.Debug("msg", "retrieved "+op)
		w.Header().Set("Content-Type", jsonContentType)
		w.Write(rawJSON)
	}
}

// StatusReportHandler creates a handler that stores the DDM status report.
func StatusReportHandler(store storage.StatusStorer, hLogger log.Logger) http.HandlerFunc {
	if store == nil || hLogger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger, enrollmentID, err := contextEnrollmentID(r, hLogger)
		if err != nil {
			ErrorAndLog(w, http.StatusBadRequest, logger, "getting enrollment id", err)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "reading body", err)
			return
		}
		unhandled, status, err := ddm.ParseStatus(bodyBytes)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "parsing status report", err)
			return
		}
		logger = logger.With(
			logkeys.DeclarationCount, len(status.Declarations),
			logkeys.ErrorCount, len(status.Errors),
			logkeys.ValueCount, len(status.Values),
		)
		for _, u := range unhandled {
			logger.Debug(logkeys.Message, "unhandled status path", "path", u)
		}
		err = store.StoreDeclarationStatus(ctx, enrollmentID, status)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "storing declaration status", err)
			return
		}
		logger.Debug(logkeys.Message, "stored declaration status")
	}
}
