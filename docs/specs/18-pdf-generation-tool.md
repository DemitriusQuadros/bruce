# Spec 18: PDF Generation Tool [BACKEND]

## Overview

Implement a `pdf_generate` tool that creates PDF documents from structured input (markdown, JSON templates, or HTML). Accepts template name, variables, and optional styling. Uses `github.com/go-echarts/go-echarts` for charts, standard `text/template` for templating, and a lightweight PDF library for rendering (e.g., `github.com/jung-kurt/gofpdf` or native Go PDF). Returns a PDF file stored temporarily and dispatched via Spec 20.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- File delivery dispatcher exists (Spec 20)

## Deliverables

**Files to Create:**
- `internal/tools/pdf/generator.go` — PDF generation logic
- `internal/tools/pdf/templates/` — built-in PDF templates (invoice, report, certificate)

**Files to Modify:**
- `internal/tools/registry.go` — register PDF generator tool
- `cmd/bruce/main.go` — instantiate PDF tool
- `config.example.yml` — document PDF tool section

## Acceptance Criteria

- [ ] Tool `pdf_generate` accepts: `template` (string, e.g., "invoice"), `variables` (object)
- [ ] Supports at least 3 built-in templates: "invoice", "report", "certificate"
- [ ] Template variables are HTML-escaped to prevent injection
- [ ] Generated PDF includes: title, footer with timestamp, page numbers
- [ ] PDF output includes basic metadata (Creator: "Bruce")
- [ ] File is stored in temp directory with 1-hour expiry
- [ ] Returns file ID and download URL (dispatched via Spec 20)
- [ ] Handles malformed templates gracefully with clear error message
- [ ] PDF generation timeout: 10 seconds
- [ ] Output file size limit: 50 MB
- [ ] Latency: p95 <3s (for typical report)

## API / Component Contract

**`pdf_generate` Schema**:
```json
{
	"name": "pdf_generate",
	"description": "Generate a PDF document from a template",
	"input_schema": {
		"type": "object",
		"properties": {
			"template": {
				"type": "string",
				"enum": ["invoice", "report", "certificate"],
				"description": "Template name"
			},
			"variables": {
				"type": "object",
				"description": "Template variables (varies by template)"
			},
			"filename": {
				"type": "string",
				"description": "Output filename (optional, defaults to template_timestamp.pdf)"
			}
		},
		"required": ["template", "variables"]
	}
}
```

**Response**:
```json
{
	"file_id": "pdf_abc123def",
	"filename": "invoice_2024-03-13.pdf",
	"url": "/api/v1/files/pdf_abc123def/download",
	"size_bytes": 45000,
	"created_at": "2024-03-13T14:02:00Z",
	"expires_at": "2024-03-13T15:02:00Z"
}
```

**`internal/tools/pdf/generator.go`**:
```go
type Generator struct {
	templates map[string]*template.Template
	tempDir   string
	expiry    time.Duration
}

func (g *Generator) Generate(ctx context.Context, templateName string, variables map[string]interface{}) (string, error) {
	// Load template
	// Validate variables
	// Render to PDF
	// Store in temp directory
	// Return file ID
}

func (g *Generator) Cleanup() {
	// Remove expired PDFs
}
```

**Built-in Template Variables**:
- **invoice**: `company_name`, `invoice_number`, `date`, `items` (array: name, qty, price), `subtotal`, `tax`, `total`, `customer_name`, `customer_email`
- **report**: `title`, `sections` (array: heading, content), `author`, `footer_text`
- **certificate**: `recipient_name`, `achievement`, `date`, `issuer_name`, `signature_placeholder`

## Out of Scope

- Custom template upload (fixed templates only for Phase 1)
- Embedded images (text/basic formatting only)
- Multi-page charts or complex layouts
- Font embedding or special characters (TTF support)
- Real-time template preview
- Digital signatures or encryption

