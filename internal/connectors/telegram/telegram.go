package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/hibiken/asynq"
)

const apiBase = "https://api.telegram.org"

// Connector connects to Telegram via long-polling and implements worker.Dispatcher.
type Connector struct {
	token       string
	httpClient  *http.Client
	asynqClient *asynq.Client
	offset      int
	stopCh      chan struct{}
}

// New creates a Telegram connector. token must be the bot token from @BotFather.
func New(token string, asynqClient *asynq.Client) *Connector {
	return &Connector{
		token:       token,
		httpClient:  &http.Client{Timeout: 45 * time.Second}, // must exceed long-poll timeout (30s)
		asynqClient: asynqClient,
		stopCh:      make(chan struct{}),
	}
}

// Start launches the background poll goroutine. It is non-blocking.
func (c *Connector) Start(ctx context.Context) error {
	log.Println("INFO: telegram: starting long-poll loop")
	go c.pollLoop(ctx)
	return nil
}

// Stop signals the poll loop to exit.
func (c *Connector) Stop() error {
	close(c.stopCh)
	return nil
}

// Send implements worker.Dispatcher. It sends a text message to the given chat.
// channelID must be the decimal string representation of the Telegram chat ID.
func (c *Connector) Send(channelID, message string) error {
	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: invalid channel_id %q: %w", channelID, err)
	}

	body, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal sendMessage: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBase, c.token)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: sendMessage POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: sendMessage status %d: %s", resp.StatusCode, raw)
	}
	log.Printf("DEBUG: telegram: sent message to chat %s", channelID)
	return nil
}

// SendFile sends a document to the given chat. Intended for future PDF delivery use-cases.
func (c *Connector) SendFile(ctx context.Context, chatID, filename string, data []byte) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("chat_id", chatID); err != nil {
		return fmt.Errorf("telegram: write chat_id field: %w", err)
	}
	part, err := w.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("telegram: create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("telegram: write file data: %w", err)
	}
	w.Close()

	url := fmt.Sprintf("%s/bot%s/sendDocument", apiBase, c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("telegram: build sendDocument request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: sendDocument POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: sendDocument status %d: %s", resp.StatusCode, raw)
	}
	return nil
}
