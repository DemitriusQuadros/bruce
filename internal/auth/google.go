// Package auth provides OAuth 2.0 authentication handlers for Google Workspace.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleScopes defines the required Google OAuth 2.0 scopes.
var GoogleScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/calendar",
	"https://www.googleapis.com/auth/documents",
	"https://www.googleapis.com/auth/spreadsheets",
	"https://www.googleapis.com/auth/drive",
}

// GoogleHandler holds OAuth 2.0 configuration and state for Google Workspace authentication.
type GoogleHandler struct {
	config     *oauth2.Config
	tokenRepo  *SQLiteOAuthTokenRepository
	stateMutex sync.Mutex
	stateMap   map[string]time.Time // state string -> expiry time
}

// NewGoogleHandler creates a new GoogleHandler with the given OAuth credentials.
func NewGoogleHandler(clientID, clientSecret, redirectURI string, db *sql.DB) *GoogleHandler {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       GoogleScopes,
		Endpoint:     google.Endpoint,
	}

	return &GoogleHandler{
		config:    config,
		tokenRepo: NewOAuthTokenRepository(db),
		stateMap:  make(map[string]time.Time),
	}
}

// StartHandler generates a random state, stores it with a 10-minute TTL,
// and returns an HTTP 302 redirect to the Google OAuth consent screen.
func (h *GoogleHandler) StartHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate random 32-byte hex state.
		stateBytes := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, stateBytes); err != nil {
			http.Error(w, "failed to generate state", http.StatusInternalServerError)
			return
		}
		state := hex.EncodeToString(stateBytes)

		// Store state with 10-minute TTL.
		h.stateMutex.Lock()
		h.stateMap[state] = time.Now().Add(10 * time.Minute)
		h.stateMutex.Unlock()

		// Clean up expired states.
		h.cleanupExpiredStates()

		// Build and redirect to Google OAuth consent screen.
		authURL := h.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// CallbackHandler validates the state parameter, exchanges the auth code for a token,
// and persists the token to the database.
func (h *GoogleHandler) CallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")

		// Check for OAuth errors from Google.
		if errParam != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":%q}`, errParam)
			return
		}

		// Validate state parameter.
		if !h.validateAndConsumeState(state) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"invalid or expired state"}`)
			return
		}

		// Exchange code for token.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		token, err := h.config.Exchange(ctx, code)
		if err != nil {
			log.Printf("oauth: token exchange failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"failed to exchange code"}`)
			return
		}

		// Persist token to database (userID = "default" for single-user MVP).
		if err := h.tokenRepo.UpsertToken("default", "google", token); err != nil {
			log.Printf("oauth: failed to persist token: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"failed to save credentials"}`)
			return
		}

		// Redirect to success page.
		http.Redirect(w, r, "/#settings?oauth=success", http.StatusFound)
	}
}

// OAuthConfig returns the underlying oauth2.Config used by this handler.
// Tool implementations use this to build a TokenSource for Google API clients.
func (h *GoogleHandler) OAuthConfig() *oauth2.Config {
	return h.config
}

// GetToken retrieves the token for the given userID and provider from the database,
// and refreshes it if expired.
func (h *GoogleHandler) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	// Read token from database.
	token, err := h.tokenRepo.GetToken(userID, "google")
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	// Check if token is expired and refresh if necessary.
	if token.Expiry.Before(time.Now()) {
		// Use the TokenSource to refresh the token automatically.
		src := h.config.TokenSource(ctx, token)
		refreshed, err := src.Token()
		if err != nil {
			return nil, fmt.Errorf("refresh token: %w", err)
		}

		// Persist refreshed token if it changed.
		if refreshed.AccessToken != token.AccessToken {
			if err := h.tokenRepo.UpsertToken(userID, "google", refreshed); err != nil {
				log.Printf("WARNING: failed to persist refreshed token: %v", err)
			}
		}

		return refreshed, nil
	}

	return token, nil
}

// validateAndConsumeState checks if the state is present in the map and not expired,
// then deletes it (one-time use).
func (h *GoogleHandler) validateAndConsumeState(state string) bool {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()

	expiry, exists := h.stateMap[state]
	if !exists {
		return false
	}

	// Check if state has expired.
	if time.Now().After(expiry) {
		delete(h.stateMap, state)
		return false
	}

	// Consume the state (one-time use).
	delete(h.stateMap, state)
	return true
}

// cleanupExpiredStates removes all expired states from the map.
func (h *GoogleHandler) cleanupExpiredStates() {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()

	now := time.Now()
	for state, expiry := range h.stateMap {
		if now.After(expiry) {
			delete(h.stateMap, state)
		}
	}
}
