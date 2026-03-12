package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"bruce/internal/domain"
	"bruce/internal/repository"
)

// sessionResponse represents a session in the API response.
type sessionResponse struct {
	ID            string `json:"id"`
	ConnectorType string `json:"connector_type"`
	ChannelID     string `json:"channel_id"`
	SystemPrompt  string `json:"system_prompt"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// SessionsHandler godoc
// @Summary      Manage sessions
// @Description  Create, list, retrieve, update, and delete chat sessions
// @Tags         sessions
// @Produce      json
// @Success      200  {object}  []sessionResponse
// @Router       /api/v1/sessions [get]
// @Router       /api/v1/sessions/{id} [get]
// @Router       /api/v1/sessions/{id} [patch]
func SessionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "session repository not initialized")
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		switch r.Method {
		case http.MethodGet:
			if id == "" {
				handleListSessions(w, r, repo)
			} else {
				handleGetSession(w, r, repo, id)
			}
		case http.MethodPatch:
			handlePatchSession(w, r, repo, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// handleListSessions returns all sessions ordered by updated_at DESC.
func handleListSessions(w http.ResponseWriter, r *http.Request, repo repository.SessionRepository) {
	sessions, err := repo.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch sessions")
		return
	}

	if sessions == nil {
		sessions = []*domain.Session{}
	}

	result := make([]sessionResponse, len(sessions))
	for i, s := range sessions {
		result[i] = sessionToResponse(s)
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGetSession returns a single session by ID.
func handleGetSession(w http.ResponseWriter, r *http.Request, repo repository.SessionRepository, id string) {
	s, err := repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, sessionToResponse(s))
}

// handlePatchSession partially updates a session.
func handlePatchSession(w http.ResponseWriter, r *http.Request, repo repository.SessionRepository, id string) {
	var req struct {
		SystemPrompt *string `json:"system_prompt"`
		IsActive     *bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check that the session exists first.
	s, err := repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Apply partial updates.
	if req.SystemPrompt != nil {
		if err := repo.UpdateSystemPrompt(id, *req.SystemPrompt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update system prompt")
			return
		}
		s.SystemPrompt = *req.SystemPrompt
	}

	if req.IsActive != nil {
		if err := repo.SetActive(id, *req.IsActive); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update active status")
			return
		}
		s.IsActive = *req.IsActive
	}

	// Update the updated_at timestamp (fetch fresh).
	s, err = repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated session")
		return
	}

	writeJSON(w, http.StatusOK, sessionToResponse(s))
}

// sessionToResponse converts a domain.Session to sessionResponse for API output.
func sessionToResponse(s *domain.Session) sessionResponse {
	return sessionResponse{
		ID:            s.ID,
		ConnectorType: s.ConnectorType,
		ChannelID:     s.ChannelID,
		SystemPrompt:  s.SystemPrompt,
		IsActive:      s.IsActive,
		CreatedAt:     s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
