package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"bruce/internal/domain"
	"bruce/internal/repository"
)

// messageResponse represents a message in the API response.
type messageResponse struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// MessagesHandler godoc
// @Summary      Session messages
// @Description  List messages within a session
// @Tags         messages
// @Produce      json
// @Success      200  {object}  []messageResponse
// @Router       /api/v1/sessions/{id}/messages [get]
func MessagesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messageRepo, ok := r.Context().Value("messageRepo").(repository.MessageRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "message repository not initialized")
			return
		}

		sessionRepo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "session repository not initialized")
			return
		}

		vars := mux.Vars(r)
		sessionID := vars["id"]

		switch r.Method {
		case http.MethodGet:
			handleGetMessages(w, r, sessionRepo, messageRepo, sessionID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// handleGetMessages returns messages for a session with optional pagination.
func handleGetMessages(w http.ResponseWriter, r *http.Request, sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository, sessionID string) {
	// Check that the session exists.
	_, err := sessionRepo.GetByID(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Parse query parameters.
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 200 {
				l = 200 // max 200
			}
			limit = l
		}
	}

	// Get messages (newest first).
	messages, err := messageRepo.GetRecent(sessionID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch messages")
		return
	}

	if messages == nil {
		messages = []*domain.Message{}
	}

	result := make([]messageResponse, len(messages))
	for i, m := range messages {
		result[i] = messageResponse{
			ID:        m.ID,
			SessionID: m.SessionID,
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, result)
}
