# Git (Local) Tools

Four tools for status, commit, push, and branch operations on local git repositories. Bruce runs `git` as a subprocess — SSH keys or a credential helper must already be configured at the OS level.

## Config snippet

```yaml
tools:
  git_local:
    enabled: true
    home_dir: "/Users/me/projects"   # base directory for relative repo paths
    timeout_seconds: 30
```

`home_dir` acts as the root for all `repo_path` values. If `repo_path` is `"api"`, Bruce looks for the repo at `{home_dir}/api`.

## SSH / credential assumption

Bruce inherits the environment of the process that started it. Git operations that require authentication (push, fetch) depend on credentials already present in that environment:

- **SSH keys:** The SSH agent must be running and have the appropriate key loaded (`ssh-add`).
- **HTTPS credentials:** A `git credential.helper` (e.g. macOS Keychain, `git-credential-osxkeychain`) must be configured at the OS level.

Bruce does not store or inject Git credentials. If a push fails with `permission denied (publickey)`, the issue is with the OS-level SSH configuration, not with Bruce.

## Tools reference

| Tool | Description |
|---|---|
| `git_status` | Show working tree status for a repo |
| `git_commit` | Stage all changes and create a commit |
| `git_push` | Push the current branch to origin |
| `git_branch` | List, create, or delete branches |

**`git_status` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo_path` | string | Yes | Path to the repo relative to `home_dir` |

Returns: `branch` (current branch name), `staged` (array of filenames), `unstaged` (array of filenames), `untracked` (array of filenames).

**`git_commit` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo_path` | string | Yes | Path to the repo relative to `home_dir` |
| `message` | string | Yes | Commit message |

Runs `git add -A && git commit -m "{message}"` — all tracked and untracked files are staged.

Returns: `hash` (short commit SHA), `message`, `committed_at`.

**`git_push` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo_path` | string | Yes | Path to the repo relative to `home_dir` |
| `branch` | string | No | Branch to push (defaults to current branch) |

Pushes to `origin`. Returns: `branch`, `pushed_at`.

**`git_branch` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo_path` | string | Yes | Path to the repo relative to `home_dir` |
| `action` | string | Yes | `list` \| `create` \| `delete` |
| `branch_name` | string | No | Branch name (required for `create` and `delete`) |

Returns for `list`: array of branch names. Returns for `create`/`delete`: `branch_name`, `action`.

## Example prompts

- "What's the git status of my api repo?"
- "Commit all changes in the frontend repo with message 'fix navigation bug'"
- "Push the main branch of my backend repo"
- "List all branches in the api repo"
- "Create a branch called feature/new-login in the frontend repo"

## Limitations

- `git_commit` always runs `git add -A` — there is no way to stage a subset of files via this tool. Use the [bash tool](bash.md) for selective staging.
- `git_push` pushes to `origin` only — no support for pushing to other remotes.
- No cherry-pick, rebase, merge, or stash operations.
- No credential storage — Bruce does not prompt for passwords or tokens.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `permission denied (publickey)` on push | SSH key not loaded or not authorized | Run `ssh-add ~/.ssh/id_ed25519` (or your key path) before starting Bruce |
| Push times out on large repos | `timeout_seconds` too low | Increase `timeout_seconds` in `config.yml` (e.g. to 60 or 120) |
| "not a git repository" | `repo_path` does not point to a git repo | Verify the path relative to `home_dir` — run `git status` manually to confirm |
| `git_commit` succeeds but nothing committed | Working tree was already clean | Check `git_status` first; commit only when staged/unstaged files are present |
| `git_branch` delete fails | Branch is checked out or has unmerged changes | Switch to a different branch first, or merge/discard changes |
