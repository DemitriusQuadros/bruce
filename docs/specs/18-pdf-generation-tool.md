# Spec 18: PDF Generation Tool [BACKEND]

## Overview

Implement `pdf_generate` tool that converts Markdown or HTML to PDF. Uses `chromedp` (headless Chrome/Chromium in Go) to render HTML to PDF. Tool accepts Markdown/HTML content as input, renders it, saves to disk, and returns file path. Output file is intended for delivery via Discord/Telegram (Spec 21). Dockerfile includes `chromium-browser` for headless rendering.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- File I/O tool exists (Spec 17)
- Docker image can be extended (`Dockerfile` exists)
- `chromedp` library available

## Deliverables

**Files to Create:**
- `internal/tools/document/pdf.go` — PDF generation tool
- `internal/tools/document/renderer.go` — Markdown/HTML to PDF renderer

**Files to Modify:**
- `Dockerfile` — add `chromium-browser` package to base image
- `internal/tools/registry.go` — register PDF tool at startup
- `cmd/bruce/main.go` — instantiate document tool
- `config.example.yml` — document PDF output directory

## Acceptance Criteria

- [ ] Tool name is `pdf_generate`; schema includes: `content` (string), `format` (enum: "markdown" | "html"), `filename` (optional, auto-generated if not provided)
- [ ] Markdown is converted to HTML (via `goldmark` or similar) before rendering
- [ ] Chromium is spawned in headless mode (no UI), renders HTML, outputs PDF
- [ ] Output PDF is saved to configured directory (default: `./data/pdfs/`)
- [ ] Returns: file path, filename, size in bytes
- [ ] Tool handles Chromium startup timeout (30s max)
- [ ] Tool handles malformed HTML gracefully (renders best-effort)
- [ ] PDF output is deterministic (same input = same PDF bytes, modulo timestamps)
- [ ] Latency: <10s p95 (Chromium startup + render)
- [ ] Dockerfile build includes `chromium-browser` and `CGO_ENABLED=1` for any C-based libraries

## API / Component Contract

**Tool Schema**:
```json
{
	"name": "pdf_generate",
	"description": "Convert Markdown or HTML to PDF",
	"input_schema": {
		"type": "object",
		"properties": {
			"content": {
				"type": "string",
				"description": "Markdown or HTML content"
			},
			"format": {
				"type": "string",
				"enum": ["markdown", "html"],
				"default": "markdown"
			},
			"filename": {
				"type": "string",
				"description": "Output filename (without .pdf extension); auto-generated if not provided"
			}
		},
		"required": ["content", "format"]
	}
}
```

**`internal/tools/document/pdf.go`**:
```go
type PDFGenerator struct {
	outputDir string
}

func (g *PDFGenerator) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)

type PDFOutput struct {
	FilePath string    // Absolute path
	Filename string    // Basename
	SizeBytes int64
	MimeType string    // "application/pdf"
}
```

## Out of Scope

- Word/Excel generation
- PDF merge/split
- Password protection
- Watermarks
- Custom fonts (use system defaults)
- JavaScript in PDF (render before PDF step)
