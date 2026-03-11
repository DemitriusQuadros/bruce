package handlers

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler returns a stub HTTP handler for config endpoints.
// Full implementation is deferred to a later spec.
func ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
	}
}
