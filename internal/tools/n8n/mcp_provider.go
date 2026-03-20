package n8n

import (
	"context"
	"log"
	"sync"
	"time"

	"bruce/internal/config"
	"bruce/internal/tools"
)

// MCPProvider discovers tools from an n8n MCP endpoint and registers them
// in the tool registry. It handles reconnection with exponential backoff.
type MCPProvider struct {
	cfg       config.MCPConfig
	registry  *tools.Registry
	mu        sync.RWMutex
	toolNames map[string]string // sanitized → original
}

// NewMCPProvider creates a new MCPProvider.
func NewMCPProvider(cfg config.MCPConfig, registry *tools.Registry) *MCPProvider {
	prefix := cfg.ToolNamePrefix
	if prefix == "" {
		prefix = "n8n_mcp_"
	}
	cfg.ToolNamePrefix = prefix

	maxAttempts := cfg.MaxReconnectAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	cfg.MaxReconnectAttempts = maxAttempts

	return &MCPProvider{
		cfg:       cfg,
		registry:  registry,
		toolNames: make(map[string]string),
	}
}

// Start performs an initial tool discovery and launches a reconnect loop in the
// background. Discovery failures are non-fatal — the provider logs a warning and
// retries with exponential backoff.
func (p *MCPProvider) Start(ctx context.Context) {
	if err := p.discover(ctx); err != nil {
		log.Printf("[n8n] MCPProvider: initial discovery failed: %v — starting reconnect loop", err)
		go p.reconnectLoop(ctx)
		return
	}
	log.Printf("[n8n] MCPProvider: initial discovery succeeded")
}

// discover calls tools/list on the MCP SSE URL and registers discovered tools.
func (p *MCPProvider) discover(ctx context.Context) error {
	rawTools, err := discoverTools(ctx, p.cfg.SSEURL, p.cfg.BearerToken)
	if err != nil {
		return err
	}

	registered := 0
	for _, rawTool := range rawTools {
		originalName, _ := rawTool["name"].(string)
		if originalName == "" {
			continue
		}

		description, _ := rawTool["description"].(string)
		var schema map[string]interface{}
		if s, ok := rawTool["inputSchema"].(map[string]interface{}); ok {
			schema = s
		} else if s, ok := rawTool["input_schema"].(map[string]interface{}); ok {
			schema = s
		}

		sanitized := sanitizeName(originalName, p.cfg.ToolNamePrefix)

		p.mu.Lock()
		if _, exists := p.toolNames[sanitized]; exists {
			log.Printf("[n8n] MCPProvider: skipping tool %q — sanitized name %q already registered", originalName, sanitized)
			p.mu.Unlock()
			continue
		}
		p.toolNames[sanitized] = originalName
		p.mu.Unlock()

		mcpTool := &MCPTool{
			sanitizedName: sanitized,
			originalName:  originalName,
			description:   description,
			schema:        schema,
			provider:      p,
		}

		if err := p.registry.Register(mcpTool); err != nil {
			log.Printf("[n8n] MCPProvider: failed to register tool %q: %v", sanitized, err)
			continue
		}
		registered++
		log.Printf("[n8n] MCPProvider: registered tool %q (original: %q)", sanitized, originalName)
	}

	log.Printf("[n8n] MCPProvider: registered %d MCP tools", registered)
	return nil
}

// reconnectLoop retries discovery with exponential backoff up to MaxReconnectAttempts.
func (p *MCPProvider) reconnectLoop(ctx context.Context) {
	backoff := time.Second
	for attempt := 1; attempt <= p.cfg.MaxReconnectAttempts; attempt++ {
		select {
		case <-ctx.Done():
			log.Printf("[n8n] MCPProvider: reconnect loop cancelled")
			return
		case <-time.After(backoff):
		}

		log.Printf("[n8n] MCPProvider: reconnect attempt %d/%d", attempt, p.cfg.MaxReconnectAttempts)
		if err := p.discover(ctx); err != nil {
			log.Printf("[n8n] MCPProvider: reconnect attempt %d failed: %v", attempt, err)
			if backoff < 16*time.Second {
				backoff *= 2
			}
			continue
		}

		log.Printf("[n8n] MCPProvider: reconnect succeeded on attempt %d", attempt)
		return
	}

	log.Printf("[n8n] MCPProvider: all %d reconnect attempts exhausted — MCP tools unavailable", p.cfg.MaxReconnectAttempts)
}
