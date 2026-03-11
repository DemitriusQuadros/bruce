package handlers

import (
	"encoding/json"
	"net/http"
)

// SessionsHandler godoc
// @Summary      Manage sessions
// @Description  Create, list, retrieve, update, and delete chat sessions
// @Tags         sessions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/sessions [get]
// @Router       /api/sessions [post]
// @Router       /api/sessions/{id} [get]
// @Router       /api/sessions/{id} [put]
// @Router       /api/sessions/{id} [delete]
func SessionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
	}
}
