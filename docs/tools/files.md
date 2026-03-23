# File Tools

Two tools for reading and writing local files within a sandboxed directory. All paths are validated to stay within the configured `home_dir` — path traversal attempts are rejected before any file system access occurs.

## Config snippet

```yaml
tools:
  files:
    enabled: true
    home_dir: "/Users/me/bruce-files"   # defaults to $HOME if empty
    max_file_size: 102400               # 100 KB; applies to reads
```

If `home_dir` is left empty, it defaults to the user's `$HOME` directory. It is strongly recommended to set an explicit subdirectory to limit the blast radius of accidental writes.

## Security

- **Path validation:** Every path is resolved to an absolute path and checked to ensure it falls within `home_dir`. Any path containing `..` segments or that resolves outside the base directory is rejected.
- **Symlink safety:** Symlinks whose targets resolve outside `home_dir` are also rejected.
- **Parent directory creation:** `file_write` automatically creates missing parent directories, so writing to `reports/2026/weekly.md` works even if `reports/2026/` does not yet exist.
- **Atomic writes:** Files are written to a temporary path first, then renamed — no partial writes are visible to other processes.

## Tools reference

| Tool | Description |
|---|---|
| `file_read` | Read a file's text content |
| `file_write` | Write text content to a file |

**`file_read` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | Yes | File path relative to `home_dir` |

Returns: `content`, `path` (absolute), `size_bytes`, `truncated` (true if file exceeded `max_file_size`).

**`file_write` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | Yes | File path relative to `home_dir` |
| `content` | string | Yes | Text content to write |

Returns: `path` (absolute), `size_bytes`, `created` (boolean — true if file did not exist), `overwritten` (boolean — true if file was replaced), `parent_dirs_created` (boolean — true if directories were created).

## Example prompts

- "Read my notes.txt"
- "Write a summary of our conversation to reports/weekly.md"
- "What's in the file config/settings.json?"

## Limitations

- Read is limited to `max_file_size` (default 100 KB). Larger files are returned truncated with `truncated: true`.
- Text content only — binary files (images, executables, compressed archives) are not handled correctly.
- No delete or rename operations — to remove a file, use the [bash tool](bash.md) if enabled.
- `file_write` overwrites the file completely; there is no append mode (use `file_read` → modify → `file_write` for append-like behavior).

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| "path escapes home_dir" error | Path resolves outside the configured `home_dir` | Use a relative path that stays within `home_dir`; avoid `..` segments |
| File not found | File does not exist at the resolved path | Check the path with `file_read` against a known file; verify `home_dir` is set correctly |
| Permission denied | Bruce process lacks read/write access to the file | Check OS file permissions on `home_dir` and the target file |
| Content truncated | File exceeds `max_file_size` | Increase `max_file_size` in `config.yml` or read a specific section using the bash tool with `head`/`tail` if enabled |
