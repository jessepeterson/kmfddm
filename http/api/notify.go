package api

import (
	"net/http"

	"github.com/micromdm/nanolib/log"
)

// NotifyHandler notifies enrollment IDs.
func NotifyHandler(notifier Notifier, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := notifier.Changed(
			r.Context(),
			r.URL.Query()["declaration"],
			r.URL.Query()["set"],
			r.URL.Query()["id"],
		)
		if err != nil {
			jsonErrorAndLog(w, http.StatusInternalServerError, err, "notify changed", logger)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
