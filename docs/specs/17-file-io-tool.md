# Spec 17: File I/O Tool [BACKEND]

## Overview

Implement `file_read` and `file_write` tools for local filesystem access. Both validate paths (no `..`, no absolute paths outside home dir). `file_read` accepts relative paths and returns file contents. `file_write` accepts path, filename, content and writes to disk; requires explicit confirmation before execution (Spec 20). Implements simple safeguards to prevent path traversal and unintended writes.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Confirmation system exists (Spec 20)
- Domain models defined

## Deliverables

**Files to Create:**
- `internal/tools/files/files.go` — Tool implementations (read + write)
- `internal/tools/files/validator.go` — path validation logic

**Files to Modify:**
- `internal/tools/registry.go` — register both tools at startup
- `cmd/bruce/main.go` — instantiate files tool
- `config.example.yml` — document allowed file paths / home directory

## Acceptance Criteria

- [ ] Tool `file_read` accepts: `path` (relative to home dir)
- [ ] Tool `file_write` accepts: `path`, `content` (string); requires confirmation before execution
- [ ] Path validation: reject paths containing `..`, absolute paths, symlinks outside home
- [ ] Path validation: only allow paths under `$HOME` or configured whitelist
- [ ] `file_read` returns file contents (text only, max 100KB)
- [ ] `file_read` returns error if file not found, permission denied, or path invalid
- [ ] `file_write` checks if file exists and warns user before overwriting
- [ ] `file_write` creates parent directories if they don't exist (with confirmation)
- [ ] `file_write` is atomic (write to temp, then rename) or uses transaction-like semantics
- [ ] Both tools log all operations (path, size, success/failure)
- [ ] Latency: read <100ms p95, write <200ms p95

## API / Component Contract

**`file_read` Schema**:
```json
{
	"name": "file_read",
	"description": "Read contents of a local file",
	"input_schema": {
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Relative path from home directory (no .. or absolute paths)"
			}
		},
		"required": ["path"]
	}
}
```

**`file_write` Schema**:
```json
{
	"name": "file_write",
	"description": "Write content to a local file (requires approval)",
	"input_schema": {
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Relative path from home directory"
			},
			"content": {
				"type": "string",
				"description": "File content to write"
			}
		},
		"required": ["path", "content"]
	}
}
```

**`internal/tools/files/validator.go`**:
```go
func ValidatePath(basePath, userPath string) (string, error) {
	// Resolve to absolute path
	// Check no .. or traversal
	// Ensure under home directory
	// Return canonical absolute path
}
```

## Out of Scope

- Binary file support (text only for Phase 1)
- File deletion (write-only operations)
- Directory listing (single file ops only)
- File permissions modification
- Symlink creation
