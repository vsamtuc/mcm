package http

import (
	"net/http"

	"github.com/vsamtuc/mcm/pkg/greet"
)

// NewMux creates a new HTTP mux with health check endpoints.
// The schemaReady function is used to determine readiness status.
func NewMux(schemaReady func() bool) http.Handler {
	if schemaReady == nil {
		schemaReady = func() bool { return true }
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !schemaReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("schema not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(greet.Hello("world")))
	})
	return mux
}
