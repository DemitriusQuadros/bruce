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

// HealthHandler godoc
// @Summary      Health check
// @Description  Returns server status and uptime
// @Tags         system
// @Produce      json
// @Success      200  {object}  healthResponse
// @Router       /health [get]
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
