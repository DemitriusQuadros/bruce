package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"bruce/internal/domain"
	"bruce/internal/repository"
)

type toolExecutionResponse struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	ToolName   string `json:"tool_name"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	LatencyMS  int64  `json:"latency_ms"`
	Success    bool   `json:"success"`
	ErrorMsg   string `json:"error_msg"`
	ExecutedAt string `json:"created_at"`
}

type toolExecutionsListResponse struct {
	Executions []toolExecutionResponse `json:"executions"`
	Total      int                     `json:"total"`
}

// ToolExecutionsHandler godoc
// @Summary      Session tool executions
// @Description  List tool executions for a session
// @Tags         sessions
// @Produce      json
// @Param        id     path  string  true  "Session ID"
// @Param        limit  query int     false "Max results (default 100)"
// @Param        offset query int     false "Pagination offset"
// @Success      200    {object}  toolExecutionsListResponse
// @Router       /api/v1/sessions/{id}/tool-executions [get]
func ToolExecutionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionRepo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "session repository not initialized")
			return
		}

		toolExecRepo, ok := r.Context().Value("toolExecutionRepo").(repository.ToolExecutionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "tool execution repository not initialized")
			return
		}

		sessionID := mux.Vars(r)["id"]

		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				if v > 500 {
					v = 500
				}
				limit = v
			}
		}

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil && v >= 0 {
				offset = v
			}
		}

		var executions []*domain.ToolExecution
		var total int
		var err error

		if sessionID == "" || sessionID == "all" {
			executions, err = toolExecRepo.GetAll(limit, offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to fetch tool executions")
				return
			}
			total, err = toolExecRepo.CountAll()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to count tool executions")
				return
			}
		} else {
			if _, err := sessionRepo.GetByID(sessionID); err != nil {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			executions, err = toolExecRepo.GetBySession(sessionID, limit, offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to fetch tool executions")
				return
			}
			total, err = toolExecRepo.CountBySession(sessionID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to count tool executions")
				return
			}
		}

		resp := toolExecutionsListResponse{
			Executions: make([]toolExecutionResponse, len(executions)),
			Total:      total,
		}
		for i, e := range executions {
			resp.Executions[i] = toolExecutionResponse{
				ID:         e.ID,
				SessionID:  e.SessionID,
				ToolName:   e.ToolName,
				Input:      e.Input,
				Output:     e.Output,
				LatencyMS:  e.LatencyMS,
				Success:    e.Success,
				ErrorMsg:   e.ErrorMsg,
				ExecutedAt: e.ExecutedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
