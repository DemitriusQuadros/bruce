# Spec 16: Bash Execution Tool [BACKEND]

## Overview

Implement a `bash_exec` tool that gives the AI model the ability to run arbitrary bash commands on the host machine within operator-defined security boundaries. This tool is useful for developer workflows: file inspection, git operations, builds, environment checks, etc. Implements defense-in-depth security: allowlist/blocklist, path traversal prevention, stdin isolation, limited environment, output truncation, and timeout with graceful termination.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- No external dependencies required beyond stdlib

## Deliverables

**Files to Create:**
- `internal/tools/bash/bash.go` — `BashTool` struct implementing `tools.Tool`; subprocess management, timeout/signal logic
- `internal/tools/bash/validator.go` — `ValidateCommand()` (allowlist + blocklist) and `ValidateWorkingDir()` (path traversal prevention)

**Files to Modify:**
- `internal/config/config.go` — Add `BashConfig` struct; populate `ToolsConfig.Bash`
- `config.example.yml` — Add `tools.bash` section with all config keys documented
- `cmd/bruce/main.go` — Instantiate `bash.New(cfg.Tools.Bash)` and register if `Enabled`

## Acceptance Criteria

- [ ] Tool `bash_exec` accepts: `command` (string, required), `working_dir` (string, optional), `timeout_seconds` (integer, optional)
- [ ] Response includes: `stdout`, `stderr`, `exit_code`, `truncated`, `working_dir`
- [ ] Non-zero exit code is reported as `exit_code` in response; only infrastructure failures return Go errors
- [ ] **Allowlist**: if `allowed_commands` is non-empty, only listed binary names are permitted
- [ ] **Blocklist** (always active): reject `rm -rf /`, fork bombs (`:(){ :|:&`, `(){ (|.&` ), pipe-to-shell (`| bash`, `| sh`), raw device redirects (`> /dev/`, `< /dev/`), `eval $(` patterns
- [ ] **Working dir**: resolved via `filepath.EvalSymlinks`; must remain under configured base dir; reject `..` traversal
- [ ] **Stdin**: connected to `/dev/null` (no interactive input)
- [ ] **Environment**: only `PATH`, `HOME`, `USER`, `LANG` passed to subprocess
- [ ] **Output cap**: combined stdout+stderr truncated at `max_output_bytes` (default 64KB); `truncated` flag set if capped
- [ ] **Timeout**: SIGTERM → 3s grace period → SIGKILL; timeout honored per request or config max (default 30s, hard cap 60s)
- [ ] Path validation: no `..`, no symlink escapes, must remain under base dir
- [ ] Command validation is applied before subprocess spawn
- [ ] All executions logged to `tool_executions` table (success/failure)
- [ ] Latency: p95 <3s for simple commands (varies by command)

## API / Component Contract

**`bash_exec` Schema**:
```json
{
	"name": "bash_exec",
	"description": "Execute a bash command with restrictions and timeout",
	"input_schema": {
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "Bash command to execute (passed to bash -c)"
			},
			"working_dir": {
				"type": "string",
				"description": "Relative subdirectory under configured base (no .. allowed)"
			},
			"timeout_seconds": {
				"type": "integer",
				"description": "Command timeout in seconds (capped at config max, default 30)"
			}
		},
		"required": ["command"]
	}
}
```

**Response Format**:
```json
{
	"stdout": "file1.txt\nfile2.txt\n",
	"stderr": "",
	"exit_code": 0,
	"truncated": false,
	"working_dir": "/home/user"
}
```

**Error Responses** (validation failures return Go errors):
- Invalid command (blocked pattern): `"validation failed: command contains forbidden pattern"`
- Invalid working dir (path traversal): `"validation failed: working_dir escapes base directory"`
- Command not in allowlist: `"validation failed: command not in allowed list"`
- Timeout: `"tool execution timed out"` (from executor.go wrapper)

**`internal/tools/bash/bash.go`**:
```go
type BashTool struct {
	config BashConfig
}

type BashConfig struct {
	Enabled         bool     // Enable tool
	AllowedCommands []string // If non-empty, only these binaries allowed
	WorkingDir      string   // Base directory (absolute path)
	MaxOutputBytes  int      // Truncate stdout+stderr at this size
	TimeoutSeconds  int      // Default timeout for commands
}

func (bt *BashTool) Name() string { return "bash_exec" }
func (bt *BashTool) Definition() ai.ToolDefinition { /* ... */ }
func (bt *BashTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
```

**`internal/tools/bash/validator.go`**:
```go
func ValidateCommand(cmd string, allowlist []string) error {
	// Check allowlist if non-empty
	// Check blocklist (always):
	//   - rm -rf /
	//   - :(){ :|:& and (){ (|.&  (fork bombs)
	//   - | bash, | sh (pipe to shell)
	//   - > /dev/, < /dev/ (device redirects)
	//   - eval $( (eval injection)
	// Return error if blocked
}

func ValidateWorkingDir(baseDir, relDir string) (string, error) {
	// Resolve relDir via filepath.EvalSymlinks
	// Ensure no .. traversal
	// Ensure result stays under baseDir
	// Return canonical absolute path or error
}
```

## Configuration Schema

```yaml
tools:
  bash:
    enabled: false
    allowed_commands: []        # If empty, all commands allowed (except blocklist)
    working_dir: "/tmp/bruce"   # Base directory for relative paths
    max_output_bytes: 65536     # 64KB default; truncate stdout+stderr at this size
    timeout_seconds: 30         # Default command timeout; capped at 60s hard max
```

## Security Model

**Defense-in-depth**:

1. **Allowlist** (optional): If `allowed_commands` is non-empty, only binary names in the list may be invoked. Binary name is extracted from the command (first word after stripping whitespace).

2. **Blocklist** (always active): Reject:
   - `rm -rf /` (and similar destructive patterns)
   - Fork bombs: `:(){ :|:&` and `(){ (|.&`
   - Pipe-to-shell: `| bash`, `| sh`, `|bash`, `|sh`
   - Raw device redirects: `> /dev/`, `< /dev/`
   - Eval injection: `eval $(` patterns

3. **Working directory**: Resolved via `filepath.EvalSymlinks` to eliminate symlink escapes. Must remain under configured base dir.

4. **Stdin**: Connected to `/dev/null` — no interactive input allowed.

5. **Environment**: Only `PATH`, `HOME`, `USER`, `LANG` are passed. All others stripped.

6. **Output truncation**: Combined stdout+stderr capped at `max_output_bytes` (default 64KB). `truncated: true` flag is set if output was capped.

7. **Timeout**:
   - Default per-command: `timeout_seconds` from config or request
   - Hard cap: 60 seconds
   - Behavior: SIGTERM → 3s grace → SIGKILL
   - Wrapped by executor.go 5s timeout; bash tool must complete within that (or executor timeout increases)

**Key invariant**: Non-zero exit code is NOT a Go error — it is reported as `exit_code` in the response JSON. Only validation failures, spawn errors, or timeout/signal terminations return Go errors.

## Out of Scope

- Interactive shell sessions (stdin from `/dev/null`)
- Privilege escalation (runs as same user as Bruce process)
- Job control (background processes, `&`, `wait`)
- Piping between multiple commands (single `bash -c` invocation)
- Shell builtins that spawn subshells (e.g., `source`, `.`, `exec`)

## Example Usage

**Request**:
```json
{
	"command": "ls -la",
	"working_dir": "projects",
	"timeout_seconds": 5
}
```

**Response** (exit code 0, success):
```json
{
	"stdout": "total 48\ndrwxr-xr-x  5 user staff   160 Mar 14 10:30 .\n...",
	"stderr": "",
	"exit_code": 0,
	"truncated": false,
	"working_dir": "/home/user/projects"
}
```

**Response** (exit code 1, not found):
```json
{
	"stdout": "",
	"stderr": "ls: cannot access nonexistent: No such file or directory",
	"exit_code": 1,
	"truncated": false,
	"working_dir": "/home/user/projects"
}
```

**Validation Error** (blocked pattern):
```
Error: "validation failed: command contains forbidden pattern (pipe to shell)"
```
