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
)

const (
	EnrollIDHeader  = "X-Enrollment-ID"
	jsonContentType = "application/json"
)

// parseDeclPath parses path to separate out the declaration type and identifier.
func parseDeclPath(r *http.Request) (string, string, error) {
	split := strings.SplitN(r.URL.Path, "/", 2)
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid path element count: %d", len(split))
	}
	if split[0] == "" || split[1] == "" {
		return "", "", errors.New("empty type or identifier path elements")
	}
	return split[0], split[1], nil
}

type DeclarationRetriever interface {
	RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error)
}

func ErrorAndLog(w http.ResponseWriter, status int, logger log.Logger, msg string, err error) {
	logger.Info("msg", msg, "err", err)
	http.Error(w, http.StatusText(status), status)
}

// DeclarationHandler creates a handler that fetches and returns a single declaration.
// The request URL path is assumed to contain the declaration type and identifier.
// This probably requires the handler to have the path prefix stripped before use.
func DeclarationHandler(store DeclarationRetriever, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		declarationType, declarationID, err := parseDeclPath(r)
		if err != nil {
			ErrorAndLog(w, http.StatusBadRequest, logger, "parsing path", err)
			return
		}
		logger = logger.With("decl_id", declarationID, "decl_type", declarationType)
		enrollmentID := r.Header.Get(EnrollIDHeader)
		if enrollmentID == "" {
			err = errors.New("empty enrollment id")
			ErrorAndLog(w, http.StatusBadRequest, logger, "retrieve enrollment id", err)
			return
		}
		logger = logger.With("enroll_id", enrollmentID)
		var rawDecl []byte
		if store == nil {
			err = errors.New("nil store")
		} else {
			rawDecl, err = store.RetrieveEnrollmentDeclarationJSON(r.Context(), declarationID, declarationType, enrollmentID)
		}
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "retrieving declaration", err)
			return
		}
		logger.Debug("msg", "retrieved declaration")
		w.Header().Set("Content-Type", jsonContentType)
		w.Write(rawDecl)
	}
}

type TokensDeclarationItemsRetriever interface {
	RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error)
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

// TokensDeclarationItemsHandler creates a handler that fetchs and returns either
// the tokens or declaration items JSON for an erollment ID depending on tokens.
func TokensDeclarationItemsHandler(store TokensDeclarationItemsRetriever, tokens bool, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		enrollmentID := r.Header.Get(EnrollIDHeader)
		if enrollmentID == "" {
			err = errors.New("empty enrollment id")
			ErrorAndLog(w, http.StatusBadRequest, logger, "retrieve enrollment id", err)
			return
		}
		logger = logger.With("enroll_id", enrollmentID)
		var op string
		if tokens {
			op = "tokens"
		} else {
			op = "declaration items"
		}
		var rawJSON []byte
		if store == nil {
			err = errors.New("nil storage")
		} else if tokens {
			rawJSON, err = store.RetrieveTokensJSON(r.Context(), enrollmentID)
		} else {
			rawJSON, err = store.RetrieveDeclarationItemsJSON(r.Context(), enrollmentID)
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

type StatusStorage interface {
	StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error
}

// StatusReportHandler creates a handler that writes the status report
// and enrollment ID to stdout.
func StatusReportHandler(store StatusStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		enrollmentID := r.Header.Get(EnrollIDHeader)
		if enrollmentID == "" {
			err = errors.New("empty enrollment id")
			ErrorAndLog(w, http.StatusBadRequest, logger, "retrieve enrollment id", err)
			return
		}
		logger = logger.With("enroll_id", enrollmentID)
		defer r.Body.Close()
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "reading body", err)
			return
		}
		status, err := ddm.ParseStatus(bodyBytes)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "parsing status report", err)
			return
		}
		logger.Debug(
			"msg", "status report",
			"decl_count", len(status.Declarations),
			"error_count", len(status.Errors),
			"value_count", len(status.Values),
		)
		err = store.StoreDeclarationStatus(r.Context(), enrollmentID, status)
		if err != nil {
			ErrorAndLog(w, http.StatusInternalServerError, logger, "storing declaration status", err)
			return
		}
	}
}
