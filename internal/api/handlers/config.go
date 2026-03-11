package handlers

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler godoc
// @Summary      Get / update runtime config
// @Description  Returns or updates the runtime configuration
// @Tags         config
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/config [get]
// @Router       /api/config [put]
func ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
	}
}
