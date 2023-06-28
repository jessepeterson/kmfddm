package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/ctxlog"
	"github.com/jessepeterson/kmfddm/storage"
)

type DeclarationAPIStorage interface {
	RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error)
	DeleteDeclaration(ctx context.Context, declarationID string) (bool, error)
	RetrieveDeclarationSets(ctx context.Context, declarationID string) (setNames []string, err error)
	StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error)
	RetrieveDeclarations(ctx context.Context) ([]string, error)
}

// PutDeclarationHandler stores a new or overwrites an existing declaration.
func PutDeclarationHandler(store DeclarationAPIStorage, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "reading body", logger)
			return
		}
		defer r.Body.Close()
		d, err := ddm.ParseDeclaration(bodyBytes)
		if err != nil {
			jsonErrorAndLog(w, http.StatusBadRequest, err, "parsing declaration", logger)
			return
		}
		if !d.Valid() {
			err = errors.New("invalid declaration")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "parsing declaration", logger)
			return
		}
		logger = logger.With("decl_id", d.Identifier, "decl_type", d.Type)
		changed, err := store.StoreDeclaration(r.Context(), d)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "storing declaration", logger)
			return
		}
		logger.Debug("msg", "stored declaration", "changed", changed)
		status := http.StatusNotModified
		if changed {
			status = http.StatusNoContent
		}
		http.Error(w, http.StatusText(status), status)
		if changed {
			err = notifier.DeclarationChanged(r.Context(), d.Identifier)
			if err != nil {
				logger.Info("msg", "notifying", "err", err)
				return
			}
		}
	}
}

// GetDeclarationHandler retrieves a declaration by its identifier.
// The entire request URL path is assumed to contain the declaration identifier.
// This implies the handler should have the path prefix stripped before use.
func GetDeclarationHandler(store DeclarationAPIStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		declarationID := getResourceID(r)
		if declarationID == "" {
			err = errors.New("empty declaration identifier")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "validating input", logger)
			return
		}
		logger = logger.With("declaration", declarationID)
		d, err := store.RetrieveDeclaration(r.Context(), declarationID)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "retrieving declaration", logger)
			return
		}
		logger.Debug("msg", "retrieved declaration")
		w.Header().Set("Content-Type", jsonContentType)
		_, err = w.Write(d.Raw)
		if err != nil {
			logger.Info("msg", "writing response body", "err", err)
			return
		}
	}
}

// DeleteDeclarationHandler deletes a declaration by its identifier.
// The entire request URL path is assumed to contain the declaration identifier.
// This implies the handler should have the path prefix stripped before use.
func DeleteDeclarationHandler(store DeclarationAPIStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		declarationID := getResourceID(r)
		if declarationID == "" {
			err = errors.New("empty declaration identifier")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "validating input", logger)
			return
		}
		logger = logger.With("declaration", declarationID)
		changed, err := store.DeleteDeclaration(r.Context(), declarationID)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "deleting declaration", logger)
			return
		}
		logger.Debug("msg", "deleted declaration")
		status := http.StatusNoContent
		if !changed {
			status = http.StatusNotModified
		}
		w.WriteHeader(status)
		// TODO: if storage allows any kind of cascaded deletion from in-use
		// sets then we'll need to notify the associated clients.
	}
}

// GetDeclarationSetsHandler retrieves the list of sets for an declaration ID.
// The entire request URL path is assumed to contain the set name.
// This implies the handler should have the path prefix stripped before use.
func GetDeclarationSetsHandler(store DeclarationAPIStorage, logger log.Logger) http.HandlerFunc {
	return simpleJSONResourceHandler(
		logger,
		func(ctx context.Context, resource string, _ *url.URL) (interface{}, error) {
			return store.RetrieveDeclarationSets(ctx, resource)
		},
	)
}

func GetDeclarationsHandler(store DeclarationAPIStorage, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		ids, err := store.RetrieveDeclarations(r.Context())
		if err != nil {
			jsonErrorAndLog(w, 0, err, "retrieving declarations", logger)
		}
		w.Header().Set("Content-type", jsonContentType)
		err = json.NewEncoder(w).Encode(&ids)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}

// TouchDeclarationHandler touches a declaration specified by ID.
func TouchDeclarationHandler(store storage.Toucher, notifier Notifier, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		var err error
		declarationID := getResourceID(r)
		if declarationID == "" {
			err = errors.New("empty declaration identifier")
			jsonErrorAndLog(w, http.StatusBadRequest, err, "validating input", logger)
			return
		}
		logger = logger.With("declaration", declarationID)
		err = store.TouchDeclaration(r.Context(), declarationID)
		if err != nil {
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "touching declaration", logger)
			return
		}
		http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		err = notifier.DeclarationChanged(r.Context(), declarationID)
		if err != nil {
			logger.Info("msg", "notifying", "err", err)
			return
		}
	}
}
