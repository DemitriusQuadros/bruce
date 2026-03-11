package handlers

import (
	"encoding/json"
	"net/http"
)

// MessagesHandler godoc
// @Summary      Session messages
// @Description  List and post messages within a session
// @Tags         messages
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/sessions/{id}/messages [get]
// @Router       /api/sessions/{id}/messages [post]
func MessagesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
	}
}
