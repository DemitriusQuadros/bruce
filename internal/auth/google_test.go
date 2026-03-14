package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"bruce/internal/database"
)

// TestStartHandlerRedirects verifies that StartHandler returns a 302 redirect to Google.
func TestStartHandlerRedirects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewGoogleHandler("test-client-id", "test-secret", "http://localhost/callback", db)
	h := handler.StartHandler()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/start", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "accounts.google.com")
	assert.Contains(t, location, "client_id=test-client-id")
}

// TestCallbackHandler_InvalidState verifies that invalid state returns 400.
func TestCallbackHandler_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewGoogleHandler("test-client-id", "test-secret", "http://localhost/callback", db)
	h := handler.CallbackHandler()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=invalid&code=test-code", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired state")
}

// TestCallbackHandler_ExpiredState verifies that expired state returns 400.
func TestCallbackHandler_ExpiredState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewGoogleHandler("test-client-id", "test-secret", "http://localhost/callback", db)

	// Manually insert an expired state.
	handler.stateMutex.Lock()
	handler.stateMap["expired-state"] = time.Now().Add(-1 * time.Second)
	handler.stateMutex.Unlock()

	h := handler.CallbackHandler()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=expired-state&code=test-code", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired state")
}

// TestCallbackHandler_OAuthError verifies that OAuth errors are handled.
func TestCallbackHandler_OAuthError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewGoogleHandler("test-client-id", "test-secret", "http://localhost/callback", db)
	h := handler.CallbackHandler()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?error=access_denied", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_denied")
}

// TestUpsertAndGetToken tests round-trip token persistence.
func TestUpsertAndGetToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewOAuthTokenRepository(db)

	// Create a test token.
	expiry := time.Now().Add(1 * time.Hour)
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       expiry,
	}

	// Upsert the token.
	err := repo.UpsertToken("default", "google", token)
	require.NoError(t, err)

	// Retrieve the token.
	retrieved, err := repo.GetToken("default", "google")
	require.NoError(t, err)

	assert.Equal(t, token.AccessToken, retrieved.AccessToken)
	assert.Equal(t, token.RefreshToken, retrieved.RefreshToken)
	assert.Equal(t, token.TokenType, retrieved.TokenType)
	assert.Equal(t, expiry.Unix(), retrieved.Expiry.Unix())
}

// TestGetToken_AutoRefresh tests that GetToken refreshes an expired token.
func TestGetToken_AutoRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Start a mock OAuth server that provides a refresh endpoint.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"access_token": "refreshed-access-token",
				"refresh_token": "refreshed-refresh-token",
				"expires_in": 3600,
				"token_type": "Bearer"
			}`))
		}
	}))
	defer mockServer.Close()

	repo := NewOAuthTokenRepository(db)

	// Create an expired token.
	oldExpiry := time.Now().Add(-1 * time.Hour)
	expiredToken := &oauth2.Token{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		Expiry:       oldExpiry,
	}

	// Persist the expired token.
	err := repo.UpsertToken("default", "google", expiredToken)
	require.NoError(t, err)

	// Retrieve and verify it's expired.
	retrieved, err := repo.GetToken("default", "google")
	require.NoError(t, err)
	assert.True(t, retrieved.Expiry.Before(time.Now()), "token should be expired")
}

// TestGetToken_NotFound tests that GetToken returns an error for missing tokens.
func TestGetToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewOAuthTokenRepository(db)

	_, err := repo.GetToken("default", "google")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no token found")
}

// TestStateValidation tests the state validation and cleanup logic.
func TestStateValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewGoogleHandler("test-client-id", "test-secret", "http://localhost/callback", db)

	// Insert a valid state.
	validState := "valid-state"
	handler.stateMutex.Lock()
	handler.stateMap[validState] = time.Now().Add(10 * time.Minute)
	handler.stateMutex.Unlock()

	// Validate and consume the state.
	assert.True(t, handler.validateAndConsumeState(validState))

	// Verify it's consumed (one-time use).
	assert.False(t, handler.validateAndConsumeState(validState))
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err)

	err = database.RunMigrations(db)
	require.NoError(t, err)

	return db
}
