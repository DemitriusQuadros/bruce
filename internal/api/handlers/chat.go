package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

// ChatHandler returns a handler for web chat endpoints.
// It calls the LLM directly (no Asynq queue) since the web user is waiting synchronously.
func ChatHandler(registry *ai.ProviderRegistry, cfg *config.Config, toolRegistry ai.ToolRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionRepo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "session repository not initialized")
			return
		}
		messageRepo, ok := r.Context().Value("messageRepo").(repository.MessageRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "message repository not initialized")
			return
		}
		configRepo, ok := r.Context().Value("configRepo").(repository.ConfigRepository)
		if !ok {
			writeError(w, http.StatusInternalServerError, "config repository not initialized")
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		// Route by path + method.
		msgs := isMessagesPath(r)
		switch {
		case id == "" && r.Method == http.MethodGet:
			handleListChatSessions(w, sessionRepo)
		case id == "" && r.Method == http.MethodPost:
			handleCreateChatSession(w, r, sessionRepo)
		case id != "" && msgs && r.Method == http.MethodGet:
			handleGetChatMessages(w, sessionRepo, messageRepo, id)
		case id != "" && msgs && r.Method == http.MethodPost:
			handleSendChatMessage(w, r, sessionRepo, messageRepo, configRepo, registry, cfg, toolRegistry, id)
		case id != "" && !msgs && r.Method == http.MethodGet:
			handleGetChatSession(w, sessionRepo, messageRepo, id)
		case id != "" && !msgs && r.Method == http.MethodDelete:
			handleDeleteChatSession(w, sessionRepo, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func isMessagesPath(r *http.Request) bool {
	path := r.URL.Path
	return len(path) >= 9 && path[len(path)-9:] == "/messages"
}

func handleListChatSessions(w http.ResponseWriter, repo repository.SessionRepository) {
	sessions, err := repo.GetByConnectorType("web")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch chat sessions")
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

func handleCreateChatSession(w http.ResponseWriter, r *http.Request, repo repository.SessionRepository) {
	var req struct {
		Title string `json:"title"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	channelID := uuid.New().String()
	s, err := repo.Create("web", channelID, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chat session")
		return
	}
	writeJSON(w, http.StatusCreated, sessionToResponse(s))
}

func handleGetChatSession(w http.ResponseWriter, sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository, id string) {
	s, err := sessionRepo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if s.ConnectorType != "web" {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	messages, err := messageRepo.GetContextWindow(id, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch messages")
		return
	}
	if messages == nil {
		messages = []*domain.Message{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session":  sessionToResponse(s),
		"messages": messagesToResponse(messages),
	})
}

func handleDeleteChatSession(w http.ResponseWriter, repo repository.SessionRepository, id string) {
	s, err := repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if s.ConnectorType != "web" {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if err := repo.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleGetChatMessages(w http.ResponseWriter, sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository, id string) {
	s, err := sessionRepo.GetByID(id)
	if err != nil || s.ConnectorType != "web" {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	messages, err := messageRepo.GetContextWindow(id, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch messages")
		return
	}
	if messages == nil {
		messages = []*domain.Message{}
	}
	writeJSON(w, http.StatusOK, messagesToResponse(messages))
}

func handleSendChatMessage(
	w http.ResponseWriter,
	r *http.Request,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	configRepo repository.ConfigRepository,
	llm *ai.ProviderRegistry,
	cfg *config.Config,
	toolRegistry ai.ToolRegistry,
	sessionID string,
) {
	// 1. Validate session.
	session, err := sessionRepo.GetByID(sessionID)
	if err != nil || session.ConnectorType != "web" {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// 2. Parse request body.
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// 3. Insert user message.
	userMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Content,
		Timestamp: time.Now().UTC(),
	}
	if err := messageRepo.Insert(userMsg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}

	// 4. Resolve system prompt: session → DB config → YAML default.
	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		dbPrompt, err := configRepo.Get("ui.default_system_prompt")
		if err != nil || dbPrompt == "" {
			systemPrompt = cfg.UI.DefaultSystemPrompt
		} else {
			systemPrompt = dbPrompt
		}
	}

	// 5. Fetch context window.
	historyPtrs, err := messageRepo.GetContextWindow(sessionID, cfg.Claude.ContextWindow)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch context")
		return
	}
	history := make([]domain.Message, len(historyPtrs))
	for i, m := range historyPtrs {
		history[i] = *m
	}

	// 6. Call LLM — use agent loop when tools are available, plain generation otherwise.
	ctx := ai.WithSessionID(r.Context(), sessionID)
	var response string
	var llmErr error
	if toolRegistry != nil {
		response, llmErr = ai.RunAgentLoop(ctx, llm, toolRegistry, systemPrompt, history, 10, nil)
	} else {
		response, llmErr = llm.GenerateResponse(ctx, systemPrompt, history)
	}
	if llmErr != nil {
		if errors.Is(llmErr, ai.ErrRateLimited) {
			writeError(w, http.StatusTooManyRequests, "rate limited — please try again")
			return
		}
		if errors.Is(llmErr, ai.ErrProviderDown) {
			writeError(w, http.StatusServiceUnavailable, "AI provider unavailable")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to generate response")
		return
	}

	// 7. Insert assistant message.
	assistantMsg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now().UTC(),
	}
	if err := messageRepo.Insert(assistantMsg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save response")
		return
	}

	// 8. Auto-generate title from first user message if session has no title.
	if session.Title == "" {
		title := req.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		sessionRepo.UpdateTitle(sessionID, title)
		session.Title = title
	}

	// Resolve provider name for the response.
	providerName := llm.ResolveProviderName(ctx)

	// Refetch session for updated timestamps.
	session, _ = sessionRepo.GetByID(sessionID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  messageToResponse(assistantMsg),
		"session":  sessionToResponse(session),
		"provider": providerName,
	})
}

// messageToResponse converts a domain.Message to messageResponse for API output.
func messageToResponse(m *domain.Message) messageResponse {
	return messageResponse{
		ID:        m.ID,
		SessionID: m.SessionID,
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: m.Timestamp.Format("2006-01-02T15:04:05Z"),
	}
}

// messagesToResponse converts a slice of domain.Message pointers to API responses.
func messagesToResponse(msgs []*domain.Message) []messageResponse {
	result := make([]messageResponse, len(msgs))
	for i, m := range msgs {
		result[i] = messageToResponse(m)
	}
	return result
}
