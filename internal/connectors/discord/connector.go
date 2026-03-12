// Package discord provides a Discord connector for Bruce.
package discord

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hibiken/asynq"

	"bruce/internal/worker"
)

// DiscordConnector connects to Discord via the discordgo SDK and implements worker.Dispatcher.
type DiscordConnector struct {
	session     *discordgo.Session
	asynqClient *asynq.Client
	botUserID   string // populated after READY event
}

// New creates a new Discord connector with the given bot token.
// Returns an error if the token is empty or discordgo initialization fails.
func New(token string, asynqClient *asynq.Client) (*DiscordConnector, error) {
	if token == "" {
		return nil, fmt.Errorf("discord bot token is required")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discordgo init: %w", err)
	}

	// Only request the intents we need — principle of least privilege
	dg.Identify.Intents = discordgo.IntentsDirectMessages |
		discordgo.IntentsDirectMessageReactions

	return &DiscordConnector{
		session:     dg,
		asynqClient: asynqClient,
	}, nil
}

// Session returns the underlying discordgo.Session for testing and inspection.
func (c *DiscordConnector) Session() *discordgo.Session {
	return c.session
}

// Connect opens the Discord WebSocket connection and registers event handlers.
func (c *DiscordConnector) Connect() error {
	c.session.AddHandler(c.handleMessage)
	c.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		c.botUserID = r.User.ID
		log.Printf("Discord connected as %s#%s", r.User.Username, r.User.Discriminator)
	})

	if err := c.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	return nil
}

// Disconnect closes the Discord WebSocket connection gracefully.
func (c *DiscordConnector) Disconnect() {
	c.session.Close()
}

// handleMessage is the MessageCreate event handler for incoming Discord DMs.
func (c *DiscordConnector) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("DEBUG: MessageCreate event received - channel=%s, author=%s, content_len=%d",
		m.ChannelID, m.Author.ID, len(m.Content))

	// Ignore messages from the bot itself
	if m.Author.ID == c.botUserID {
		log.Printf("DEBUG: ignoring message from bot itself - bot_id=%s", c.botUserID)
		return
	}

	// Phase 1: Direct Messages only
	// A DM channel has Type == discordgo.ChannelTypeDM
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Channel not in state cache — fetch it
		log.Printf("DEBUG: channel not in state cache, fetching from API - channel=%s", m.ChannelID)
		channel, err = s.Channel(m.ChannelID)
		if err != nil {
			log.Printf("ERROR: failed to fetch channel %s: %v", m.ChannelID, err)
			return
		}
	}
	if channel.Type != discordgo.ChannelTypeDM {
		log.Printf("DEBUG: ignoring non-DM message - channel=%s, type=%v", m.ChannelID, channel.Type)
		return // Ignore guild messages in Phase 1
	}

	// Ignore empty messages (attachments, embeds without text)
	content := strings.TrimSpace(m.Content)
	if content == "" {
		log.Printf("DEBUG: ignoring empty message - channel=%s", m.ChannelID)
		return
	}

	// Enqueue for async processing — identical payload structure to WhatsApp
	log.Printf("DEBUG: enqueueing task - connector=discord, channel=%s, content_len=%d, preview=%s",
		m.ChannelID, len(content), truncateForLog(content, 100))
	payload := worker.ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     m.ChannelID, // Discord DM channel ID (stable per user pair)
		Content:       content,
	}
	task, err := worker.NewProcessIncomingMessageTask(payload)
	if err != nil {
		log.Printf("ERROR: failed to create task: %v", err)
		return
	}
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		log.Printf("ERROR: failed to enqueue from channel %s: %v", m.ChannelID, err)
		return
	}
	log.Printf("DEBUG: task enqueued successfully - channel=%s", m.ChannelID)
}

// truncateForLog returns a shortened version of s for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// Send implements worker.Dispatcher.
// It sends a message to the Discord channel, chunking if necessary to respect the 2000 character limit.
func (c *DiscordConnector) Send(channelID string, message string) error {
	log.Printf("DEBUG: Discord Send called - channel=%s, message_len=%d", channelID, len(message))

	// Discord message length limit: 2000 characters
	// Claude max_tokens: 1024 ≈ ~4000 chars worst case — must chunk
	chunks := chunkMessage(message, 1900) // 1900 to leave room for formatting
	log.Printf("DEBUG: message chunked into %d chunks", len(chunks))

	for i, chunk := range chunks {
		log.Printf("DEBUG: sending chunk %d/%d - len=%d", i+1, len(chunks), len(chunk))
		_, err := c.session.ChannelMessageSend(channelID, chunk)
		if err != nil {
			// Check for rate limit (HTTP 429)
			if restErr, ok := err.(*discordgo.RESTError); ok {
				if restErr.Response.StatusCode == 429 {
					// discordgo handles rate limits internally — this shouldn't happen
					// but if it does, log and continue
					log.Printf("WARN: discord rate limited on channel %s, retrying after 1s", channelID)
					time.Sleep(1 * time.Second)
					_, err = c.session.ChannelMessageSend(channelID, chunk)
				}
			}
			if err != nil {
				log.Printf("ERROR: failed to send chunk %d to Discord channel %s: %v", i+1, channelID, err)
				return fmt.Errorf("discord send to %s: %w", channelID, err)
			}
			log.Printf("DEBUG: chunk %d sent successfully", i+1)
		} else {
			log.Printf("DEBUG: chunk %d sent successfully", i+1)
		}
	}
	log.Printf("DEBUG: all chunks sent successfully to Discord channel %s", channelID)
	return nil
}

// chunkMessage splits a message into chunks of maxLen characters,
// breaking on word boundaries where possible.
func chunkMessage(msg string, maxLen int) []string {
	if len(msg) <= maxLen {
		return []string{msg}
	}
	var chunks []string
	for len(msg) > maxLen {
		split := maxLen
		// Walk back to find a space (checking position before split)
		for split > 0 && msg[split-1] != ' ' {
			split--
		}
		if split == 0 {
			split = maxLen // No space found, hard cut
		}
		chunks = append(chunks, msg[:split])
		msg = msg[split:]
	}
	if len(msg) > 0 {
		chunks = append(chunks, msg)
	}
	return chunks
}
