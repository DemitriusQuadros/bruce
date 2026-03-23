# Bash Tool

Execute shell commands from within Bruce conversations. The bash tool is **disabled by default** and requires explicit opt-in. Read this page before enabling it in any environment where security matters.

## Security model

The bash tool uses a layered security model:

- **Blocklist (always active):** A built-in list of dangerous patterns is always enforced regardless of config — this includes `rm -rf /`, fork bombs, piping to shell interpreters, redirects to block devices, and `eval`-style injection.
- **Allowlist (optional):** If `allowed_commands` is non-empty, only the listed binary names may be executed. Any command whose first token is not in the list is rejected before execution.
- **Restricted environment:** Subprocesses inherit only `PATH`, `HOME`, `USER`, and `LANG`. Other environment variables (including secrets) are not passed to child processes.
- **No stdin:** Commands cannot read from standard input — interactive commands will hang and be killed by the timeout.
- **Output truncation:** Combined stdout+stderr is truncated at `max_output_bytes` to prevent large outputs from filling context.
- **Timeout:** Commands are given `timeout_seconds` to complete. After that, the process receives `SIGTERM`, then `SIGKILL` after a short grace period. The hard maximum is 60 seconds regardless of config.

## Config snippet

```yaml
tools:
  bash:
    enabled: true
    allowed_commands: ["git", "go", "make"]   # empty = open mode (blocklist only)
    working_dir: "/tmp/bruce"
    max_output_bytes: 65536
    timeout_seconds: 30
```

## Allowlist vs open mode

| `allowed_commands` | Behavior |
|---|---|
| `[]` (empty) | Open mode — any command not on the built-in blocklist is allowed |
| `["git", "go"]` | Strict allowlist — only `git` and `go` may be called; all other commands are rejected |

Open mode is convenient for development machines. Use strict allowlist mode for any environment where Bruce has access to sensitive files or credentials.

## Tool reference

**`bash_exec` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `command` | string | Yes | Shell command passed to `bash -c` |
| `working_dir` | string | No | Relative subdirectory within the configured `working_dir` (no `..` allowed) |
| `timeout_seconds` | integer | No | Timeout for this invocation (capped at config `timeout_seconds`; hard max 60) |

**Tool output**

| Field | Type | Description |
|---|---|---|
| `stdout` | string | Standard output from the command |
| `stderr` | string | Standard error from the command |
| `exit_code` | integer | Process exit code |
| `truncated` | boolean | `true` if output was cut at `max_output_bytes` |
| `working_dir` | string | Absolute path where the command ran |

## Example prompts

- "Run go test ./... and show me the results"
- "What files are in the current directory?"
- "Run make build and tell me if it succeeded"

## Limitations

- Interactive commands (`vim`, `less`, `ssh`, prompts) are not supported — they will block and be killed by the timeout.
- No stdin — commands that read from standard input will not receive any data.
- Maximum timeout is 60 seconds regardless of the value in config or the tool call.
- Output is truncated at `max_output_bytes` (default 64 KB); commands that produce large output will return a partial result with `truncated: true`.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| "command not allowed" error | Binary not in `allowed_commands` | Add the binary name to `allowed_commands` in `config.yml`, or switch to open mode by setting `allowed_commands: []` |
| Command times out | Execution exceeds `timeout_seconds` | Increase `timeout_seconds` (max 60) or break the command into smaller steps |
| "working_dir contains .." error | Relative path traversal attempted | Use plain subdirectory names — paths with `..` are rejected |
| Secrets missing from environment | Restricted environment strips most vars | Pass secrets explicitly as environment variables in the command string, or pre-configure them in a `.env` file read by the target program |
| Command blocked by blocklist | Command matches a built-in dangerous pattern | The built-in blocklist cannot be disabled; use a different approach that does not match the blocked pattern |
