// Package http includes handlers and utilties
package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

// ReadAllAndReplaceBody reads all of r.Body and replaces it with a new byte buffer.
func ReadAllAndReplaceBody(r *http.Request) ([]byte, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return b, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

// CORSMiddleware adds "Access-Control-Allow-" headers to the response.
// Optionally specify an origin.
func CORSMiddleware(next http.Handler, origin string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		if origin != "" {
			h.Add("Access-Control-Allow-Origin", origin)
		}
		h.Add("Access-Control-Allow-Headers", "Authorization")
		h.Add("Access-Control-Allow-Credentials", "true")
		h.Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT")
		next.ServeHTTP(w, r)
	}
}
