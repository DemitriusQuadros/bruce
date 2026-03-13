# Spec 13: Google OAuth Flow [BACKEND]

## Overview

Implement OAuth 2.0 consent flow for Google Workspace (Gmail, Calendar, Docs, Sheets, Drive). Add `/auth/google/start` and `/auth/google/callback` HTTP routes. On successful auth, exchange code for tokens, encrypt/store in SQLite `oauth_tokens` table. Tokens are retrieved at tool execution time and used to authenticate API calls. This flow is reused by all Google-based tools.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- HTTP router exists (`internal/api/router.go`)
- SQLite schema can be extended (`internal/database/schema.sql`)
- Google OAuth credentials exist (client ID, secret, redirect URI)

## Deliverables

**Files to Create:**
- `internal/auth/google.go` — OAuth flow handler, token exchange, storage
- `internal/database/migrations/001_oauth_tokens.sql` — schema for `oauth_tokens` table

**Files to Modify:**
- `internal/api/router.go` — add `/auth/google/start`, `/auth/google/callback` routes
- `internal/database/schema.sql` — add `oauth_tokens` table (or migrate)
- `internal/config/config.go` — add `google.oauth_client_id`, `google.oauth_client_secret`, `google.oauth_redirect_uri` config fields
- `config.example.yml` — document Google OAuth setup instructions

## Acceptance Criteria

- [ ] `GET /auth/google/start` returns redirect URL to Google consent screen with correct scopes (Gmail, Calendar, Docs, Sheets, Drive)
- [ ] `GET /auth/google/callback?code=...&state=...` exchanges code for tokens
- [ ] Tokens (access + refresh) are stored in `oauth_tokens` table with encrypted `access_token`
- [ ] `state` parameter is validated to prevent CSRF
- [ ] On successful auth, user is redirected to Settings page with success message
- [ ] On auth failure, error is displayed with retry link
- [ ] Tools can retrieve stored token via `auth.GetGoogleToken(user_id)` at execution time
- [ ] If token is expired, tool uses refresh token to get new access token automatically
- [ ] SQLite schema includes: `user_id`, `provider` (enum: "google", "github"), `access_token`, `refresh_token`, `expires_at`, `created_at`, `updated_at`

## API / Component Contract

**`internal/auth/google.go`**:
```go
func StartOAuthFlow(clientID, redirectURI string, scopes []string) (authURL string, state string, err error)

func ExchangeCode(ctx context.Context, code, state string, config *oauth2.Config) (*oauth2.Token, error)

func StoreToken(ctx context.Context, repo *repository.Repository, userID, token interface{}) error

func GetToken(ctx context.Context, repo *repository.Repository, userID string) (*oauth2.Token, error)

func RefreshToken(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*oauth2.Token, error)
```

**`oauth_tokens` table**:
```sql
CREATE TABLE oauth_tokens (
	id INTEGER PRIMARY KEY,
	user_id TEXT NOT NULL,
	provider TEXT NOT NULL, -- 'google', 'github'
	access_token TEXT NOT NULL,
	refresh_token TEXT,
	expires_at INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Out of Scope

- GitHub OAuth (Spec 26)
- Notion OAuth (Spec 28)
- Trello OAuth (Spec 29)
- Token encryption at rest (acceptable plaintext in SQLite for MVP)
- Multi-user token isolation (single-user MVP)
