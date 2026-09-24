package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"bruce/internal/repository"
)

type sessionSummaryResponse struct {
	SessionID    string `json:"session_id"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"message_count"`
	UpdatedAt    string `json:"updated_at"`
}

// SessionSummaryHandler handles GET /api/v1/sessions/{id}/summary
func SessionSummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionRepo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "session repository not initialized")
			return
		}

		vars := mux.Vars(r)
		sessionID := vars["id"]
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "session id is required")
			return
		}

		if _, err := sessionRepo.GetByID(sessionID); err != nil {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}

		summaryRepo, ok := r.Context().Value("sessionSummaryRepo").(repository.SessionSummaryRepository)
		if !ok || summaryRepo == nil {
			writeJSON(w, http.StatusOK, sessionSummaryResponse{
				SessionID: sessionID,
				Summary:   "",
			})
			return
		}

		summary, err := summaryRepo.Get(sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch session summary")
			return
		}

		if summary == nil {
			writeJSON(w, http.StatusOK, sessionSummaryResponse{
				SessionID: sessionID,
				Summary:   "",
			})
			return
		}

		writeJSON(w, http.StatusOK, sessionSummaryResponse{
			SessionID:    summary.SessionID,
			Summary:      summary.Summary,
			MessageCount: summary.MessageCount,
			UpdatedAt:    summary.UpdatedAt.Format(time.RFC3339),
		})
	}
}
