package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// PutDeclarationHandler returns a handler that stores a declaration.
func PutDeclarationHandler(store storage.DeclarationStorer, notifier Notifier, logger log.Logger) http.HandlerFunc {
	if store == nil || notifier == nil || logger == nil {
		panic("nil store or notifier or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "reading body", logger)
			return
		}
		d, err := ddm.ParseDeclaration(bodyBytes)
		if err != nil {
			jsonErrorAndLog(w, http.StatusBadRequest, err, "parsing declaration", logger)
			return
		}
		if !d.Valid() {
			jsonErrorAndLog(w, http.StatusBadRequest, ddm.ErrInvalidDeclaration, "parsing declaration", logger)
			return
		}
		logger = logger.With(
			logkeys.DeclarationID, d.Identifier,
			logkeys.DeclarationType, d.Type,
		)
		changed, err := store.StoreDeclaration(r.Context(), d)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "storing declaration", logger)
			return
		}
		// only notify if we have a change
		notify := changed && shouldNotify(r.URL)
		logger.Debug(
			logkeys.Message, "stored declaration",
			logkeys.Changed, changed,
			logkeys.Notify, notify,
		)
		status := http.StatusNotModified
		if changed {
			status = http.StatusNoContent
		}
		http.Error(w, http.StatusText(status), status)
		if notify {
			err = notifier.Changed(r.Context(), []string{d.Identifier}, nil, nil)
			if err != nil {
				logger.Info(logkeys.Message, "notifying", logkeys.Error, err)
				return
			}
		}
	}
}

// GetDeclarationHandler retrieves a declaration by its identifier.
// The entire request URL path is assumed to contain the declaration identifier.
// This implies the handler should have the path prefix stripped before use.
func GetDeclarationHandler(store storage.DeclarationAPIRetriever, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		declarationID := getResourceID(r)
		if declarationID == "" {
			jsonErrorAndLog(w, http.StatusBadRequest, ErrEmptyResourceID, "validating input", logger)
			return
		}
		logger = logger.With(logkeys.DeclarationID, declarationID)
		d, err := store.RetrieveDeclaration(r.Context(), declarationID)
		if err != nil {
			statusCode := 0
			if errors.Is(err, storage.ErrDeclarationNotFound) {
				statusCode = 404
			}
			jsonErrorAndLog(w, statusCode, err, "retrieving declaration", logger)
			return
		}
		logger.Debug(logkeys.Message, "retrieved declaration")
		w.Header().Set("Content-Type", jsonContentType)
		_, err = w.Write(d.Raw)
		if err != nil {
			logger.Info(logkeys.Message, "writing response body", logkeys.Error, err)
			return
		}
	}
}

// DeleteDeclarationHandler deletes a declaration by its identifier.
// We assume that any declaration deleted has no dependant delcarations
// and is not in any other sets (and so we perform no notifications).
// The entire request URL path is assumed to contain the declaration identifier.
// This implies the handler should have the path prefix stripped before use.
func DeleteDeclarationHandler(store storage.DeclarationDeleter, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		declarationID := getResourceID(r)
		if declarationID == "" {
			jsonErrorAndLog(w, http.StatusBadRequest, ErrEmptyResourceID, "validating input", logger)
			return
		}
		logger = logger.With(logkeys.DeclarationID, declarationID)
		changed, err := store.DeleteDeclaration(r.Context(), declarationID)
		if err != nil {
			jsonErrorAndLog(w, 0, err, "deleting declaration", logger)
			return
		}
		logger.Debug(logkeys.Message, "deleted declaration")
		status := http.StatusNoContent
		if !changed {
			status = http.StatusNotModified
		}
		w.WriteHeader(status)
		// TODO: if storage allows any kind of cascaded deletion from in-use
		// sets then we'll need to notify the associated clients.
	}
}

// GetDeclarationsHandler returns a handler that lists declarations.
func GetDeclarationsHandler(store storage.DeclarationsRetriever, logger log.Logger) http.HandlerFunc {
	if store == nil || logger == nil {
		panic("nil store or logger")
	}
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

// TouchDeclarationHandler modifies a declaration ServerToken specified by ID.
func TouchDeclarationHandler(store storage.Toucher, notifier Notifier, logger log.Logger) http.HandlerFunc {
	if store == nil || notifier == nil || logger == nil {
		panic("nil store or notifier or logger")
	}
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
			statusCode := 0
			if errors.Is(err, storage.ErrDeclarationNotFound) {
				statusCode = 404
			}
			jsonErrorAndLog(w, statusCode, err, "touching declaration", logger)
			return
		}
		http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		notify := shouldNotify(r.URL)
		logger.Debug("msg", "touched declaration", "notify", notify)
		if notify {
			err = notifier.Changed(r.Context(), []string{declarationID}, nil, nil)
			if err != nil {
				logger.Info("msg", "notifying", "err", err)
				return
			}
		}
	}
}
