package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"bruce/internal/worker"
)

// pollLoop runs the long-polling loop until stopCh is closed.
func (c *Connector) pollLoop(_ interface{}) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-c.stopCh:
			log.Println("INFO: telegram: poll loop stopped")
			return
		default:
		}

		updates, err := c.getUpdates()
		if err != nil {
			log.Printf("ERROR: telegram: getUpdates: %v — retrying in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-c.stopCh:
				return
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = time.Second // reset on success

		for _, u := range updates {
			c.offset = u.UpdateID + 1
			c.handleUpdate(u)
		}
	}
}

// getUpdates fetches pending updates from the Telegram Bot API using long-polling.
func (c *Connector) getUpdates() ([]Update, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?timeout=30&offset=%d", apiBase, c.token, c.offset)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET getUpdates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("getUpdates status %d: %s", resp.StatusCode, raw)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode getUpdates response: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("getUpdates returned ok=false")
	}

	var updates []Update
	if err := json.Unmarshal(apiResp.Result, &updates); err != nil {
		return nil, fmt.Errorf("unmarshal updates: %w", err)
	}
	return updates, nil
}

// handleUpdate routes a single Telegram update to the worker queue.
func (c *Connector) handleUpdate(u Update) {
	if u.Message == nil {
		return
	}
	// Phase 1: private DMs only.
	if u.Message.Chat.Type != "private" {
		return
	}
	if u.Message.Text == "" {
		return
	}

	channelID := strconv.FormatInt(u.Message.Chat.ID, 10)
	log.Printf("DEBUG: telegram: incoming message from chat %s, len=%d", channelID, len(u.Message.Text))

	payload := worker.ProcessIncomingMessagePayload{
		ConnectorType: "telegram",
		ChannelID:     channelID,
		Content:       u.Message.Text,
	}
	task, err := worker.NewProcessIncomingMessageTask(payload)
	if err != nil {
		log.Printf("ERROR: telegram: create task: %v", err)
		return
	}
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		log.Printf("ERROR: telegram: enqueue task for chat %s: %v", channelID, err)
		return
	}
	log.Printf("DEBUG: telegram: task enqueued for chat %s", channelID)
}
