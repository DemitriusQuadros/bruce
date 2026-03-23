# GitHub Tools

Four tools for repository inspection and issue management. Authentication uses a Personal Access Token (PAT) — no OAuth flow required.

## Creating a Personal Access Token

1. Go to **GitHub → Settings → Developer settings → Personal access tokens**.
2. Choose **Fine-grained tokens** (recommended) or **Tokens (classic)**.
3. For fine-grained tokens, set **Repository access** to the repos Bruce should work with and enable **Contents** (read) and **Issues** (read and write) permissions.
4. For classic tokens, select the `repo` scope.
5. Click **Generate token** and copy the value immediately — it is shown only once.

## Config snippet

```yaml
tools:
  github:
    enabled: true
    token: "ghp_your_personal_access_token"
    default_owner: "myorg"
    default_repo: "myrepo"
```

## Default owner and repo

`default_owner` and `default_repo` act as fallbacks when the LLM does not include them in a tool call. For example, if `default_owner` is `"myorg"` and `default_repo` is `"myrepo"`, the prompt "List open issues" will query `myorg/myrepo` without requiring you to name the repo every time. The LLM can override either value by providing `owner` or `repo` parameters explicitly.

## Tools reference

| Tool | Description |
|---|---|
| `github_list_branches` | List branches in a repository |
| `github_list_issues` | List issues with optional state filter |
| `github_list_prs` | List pull requests with optional state filter |
| `github_create_issue` | Create a new issue |

**`github_list_branches` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | Yes | Repository name |
| `owner` | string | No | Repository owner (falls back to `default_owner`) |
| `per_page` | integer | No | Results per page (default 30) |

Returns: `owner`, `repo`, array of branch objects with `name` and `sha`.

**`github_list_issues` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | Yes | Repository name |
| `owner` | string | No | Repository owner (falls back to `default_owner`) |
| `state` | string | No | `open` \| `closed` \| `all` (default `open`) |
| `per_page` | integer | No | Results per page (default 30) |

Returns: array of issue objects with `number`, `title`, `state`, `url`, `created_at`, `labels`.

**`github_list_prs` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | Yes | Repository name |
| `owner` | string | No | Repository owner (falls back to `default_owner`) |
| `state` | string | No | `open` \| `closed` \| `all` (default `open`) |

Returns: array of PR objects with `number`, `title`, `state`, `url`, `created_at`, `head`, `base`.

**`github_create_issue` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | Yes | Repository name |
| `owner` | string | No | Repository owner (falls back to `default_owner`) |
| `title` | string | Yes | Issue title |
| `body` | string | No | Issue body (markdown supported) |
| `labels` | array of strings | No | Label names to apply |

Returns: `number`, `url`, `created_at`.

## Example prompts

- "List open PRs in my repo"
- "Show all closed issues in the myorg/api repo"
- "Create an issue titled 'Fix login timeout' with label 'bug'"
- "What branches exist in the frontend repo?"

## Limitations

- These 4 tools cover read + issue-create only. There is no support for comments, file contents, GitHub Actions, or PR reviews.
- The token must have access to the target repositories — private repos require appropriate scopes.
- Results are not paginated beyond `per_page` — increase `per_page` (max 100 for classic tokens) to retrieve more items.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| 401 Unauthorized | Invalid or expired token | Regenerate the PAT in GitHub Settings and update `config.yml` |
| 404 Not Found | Repository does not exist or token lacks access | Verify the owner/repo and that the PAT has access to the repository |
| Rate limit exceeded (403) | Too many API calls | GitHub allows 5,000 requests/hour for authenticated PATs; wait or use a token with higher limits |
| Empty results on issues/PRs | No items match the state filter | Try `state: "all"` to confirm the repo has items |
| "default_owner is not set" | `default_owner` missing and no `owner` in prompt | Set `default_owner` in `config.yml` or always include the owner in the prompt |
