package auth

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// SQLiteOAuthTokenRepository handles persistence of OAuth tokens to SQLite.
type SQLiteOAuthTokenRepository struct {
	db *sql.DB
}

// NewOAuthTokenRepository creates a new SQLiteOAuthTokenRepository.
func NewOAuthTokenRepository(db *sql.DB) *SQLiteOAuthTokenRepository {
	return &SQLiteOAuthTokenRepository{db: db}
}

// UpsertToken inserts or replaces an OAuth token in the database.
func (r *SQLiteOAuthTokenRepository) UpsertToken(userID, provider string, token *oauth2.Token) error {
	expiresAt := int64(0)
	if !token.Expiry.IsZero() {
		expiresAt = token.Expiry.Unix()
	}

	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO oauth_tokens (user_id, provider, access_token, refresh_token, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		userID, provider, token.AccessToken, token.RefreshToken, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("upsert oauth token: %w", err)
	}
	return nil
}

// GetToken retrieves an OAuth token from the database.
func (r *SQLiteOAuthTokenRepository) GetToken(userID, provider string) (*oauth2.Token, error) {
	row := r.db.QueryRow(
		`SELECT access_token, refresh_token, expires_at FROM oauth_tokens
		 WHERE user_id = ? AND provider = ?`,
		userID, provider,
	)

	var accessToken, refreshToken string
	var expiresAt int64

	err := row.Scan(&accessToken, &refreshToken, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no token found for user %q and provider %q", userID, provider)
		}
		return nil, fmt.Errorf("get oauth token: %w", err)
	}

	// Reconstruct the oauth2.Token.
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	if expiresAt > 0 {
		token.Expiry = time.Unix(expiresAt, 0)
	}

	return token, nil
}
