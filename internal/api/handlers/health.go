// Package handlers provides HTTP handler functions for the Bruce API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// healthResponse is the JSON body returned by the /health endpoint.
type healthResponse struct {
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// HealthHandler returns an http.HandlerFunc that responds to liveness checks.
// It always returns 200 while the process is alive — no DB ping is performed.
func HealthHandler(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{
			Status:        "ok",
			UptimeSeconds: int64(time.Since(startTime).Seconds()),
		})
	}
}
