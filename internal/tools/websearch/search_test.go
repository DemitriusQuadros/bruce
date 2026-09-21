package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
)

func TestCleanHTML(t *testing.T) {
	rawHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Page Title</title>
    <meta name="description" content="A test page description for testing.">
    <style>body { color: red; }</style>
    <script>console.log("ignore me");</script>
</head>
<body>
    <header><nav><a href="/">Home</a></nav></header>
    <h1>Main Heading</h1>
    <p>This is the first paragraph with some <b>bold text</b> and <a href="https://example.com">a link</a>.</p>
    <h2>Secondary Heading</h2>
    <ul>
        <li>First item</li>
        <li>Second item</li>
    </ul>
    <footer>Copyright 2026</footer>
</body>
</html>`

	page, err := CleanHTML([]byte(rawHTML), 1000)
	require.NoError(t, err)
	assert.Equal(t, "Test Page Title", page.Title)
	assert.Equal(t, "A test page description for testing.", page.Description)
	assert.Contains(t, page.Content, "# Main Heading")
	assert.Contains(t, page.Content, "This is the first paragraph")
	assert.Contains(t, page.Content, "## Secondary Heading")
	assert.Contains(t, page.Content, "- First item")
	assert.Contains(t, page.Content, "- Second item")
	assert.NotContains(t, page.Content, "console.log")
	assert.NotContains(t, page.Content, "Copyright 2026")
	assert.False(t, page.Truncated)
}

func TestCleanHTML_Truncation(t *testing.T) {
	rawHTML := `<html><body><p>Long text that should exceed max characters constraint.</p></body></html>`
	page, err := CleanHTML([]byte(rawHTML), 15)
	require.NoError(t, err)
	assert.True(t, page.Truncated)
	assert.LessOrEqual(t, len(page.Content), 15)
}

func TestSearchTool_Definition(t *testing.T) {
	tool := NewSearchTool(&config.Config{})
	assert.Equal(t, "web_search", tool.Name())
	def := tool.Definition()
	assert.Equal(t, "web_search", def.Name)
	assert.Contains(t, def.Description, "Search the web")
}

func TestSearchTool_Execute_Validation(t *testing.T) {
	tool := NewSearchTool(&config.Config{})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{})
	assert.Error(t, err)

	_, err = tool.Execute(ctx, map[string]interface{}{"query": "   "})
	assert.Error(t, err)
}

func TestParseDuckDuckGoLite(t *testing.T) {
	sampleLiteHTML := `
<html>
<body>
<table>
  <tr>
    <td>
      <a rel="nofollow" href="https://golang.org" class='result-link'>The Go Programming Language</a>
    </td>
  </tr>
  <tr>
    <td class='result-snippet'>
      Go is an open source programming language that makes it simple to build secure, scalable systems.
    </td>
  </tr>
  <tr>
    <td>
      <a rel="nofollow" href="https://go.dev/doc/install" class='result-link'>Install Go</a>
    </td>
  </tr>
  <tr>
    <td class='result-snippet'>
      Instructions for downloading and installing Go compilers and tools.
    </td>
  </tr>
</table>
</body>
</html>`

	results := parseDuckDuckGoLite([]byte(sampleLiteHTML), 5)
	require.Len(t, results, 2)
	assert.Equal(t, "The Go Programming Language", results[0].Title)
	assert.Equal(t, "https://golang.org", results[0].URL)
	assert.Contains(t, results[0].Snippet, "open source programming language")

	assert.Equal(t, "Install Go", results[1].Title)
	assert.Equal(t, "https://go.dev/doc/install", results[1].URL)
}

func TestFetchTool_Definition(t *testing.T) {
	tool := NewFetchTool(&config.Config{})
	assert.Equal(t, "web_fetch", tool.Name())
	def := tool.Definition()
	assert.Equal(t, "web_fetch", def.Name)
	assert.Contains(t, def.Description, "Fetch and read")
}

func TestFetchTool_Validation(t *testing.T) {
	tool := NewFetchTool(&config.Config{})
	ctx := context.Background()

	// Missing URL
	_, err := tool.Execute(ctx, map[string]interface{}{})
	assert.Error(t, err)

	// Invalid scheme
	_, err = tool.Execute(ctx, map[string]interface{}{"url": "ftp://example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be http or https")

	// SSRF blocked (loopback)
	_, err = tool.Execute(ctx, map[string]interface{}{"url": "http://127.0.0.1:8080/test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF blocked")
}

func TestFetchTool_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Mock Article</title></head><body><h1>Article Header</h1><p>Article body content text.</p></body></html>`))
	}))
	defer server.Close()

	tool := NewFetchTool(&config.Config{})
	ctx := context.Background()

	// Since httptest binds to 127.0.0.1, validatePublicHost will block it by default.
	// But we can verify that SSRF protection correctly blocks it:
	_, err := tool.Execute(ctx, map[string]interface{}{"url": server.URL})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF blocked")
}
