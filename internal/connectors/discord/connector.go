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
	// Ignore messages from the bot itself
	if m.Author.ID == c.botUserID {
		return
	}

	// Phase 1: Direct Messages only
	// A DM channel has Type == discordgo.ChannelTypeDM
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Channel not in state cache — fetch it
		channel, err = s.Channel(m.ChannelID)
		if err != nil {
			log.Printf("discord: failed to fetch channel %s: %v", m.ChannelID, err)
			return
		}
	}
	if channel.Type != discordgo.ChannelTypeDM {
		return // Ignore guild messages in Phase 1
	}

	// Ignore empty messages (attachments, embeds without text)
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	// Enqueue for async processing — identical payload structure to WhatsApp
	payload := worker.ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     m.ChannelID, // Discord DM channel ID (stable per user pair)
		Content:       content,
	}
	task, err := worker.NewProcessIncomingMessageTask(payload)
	if err != nil {
		log.Printf("discord: failed to create task: %v", err)
		return
	}
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		log.Printf("discord: failed to enqueue from channel %s: %v", m.ChannelID, err)
	}
}

// Send implements worker.Dispatcher.
// It sends a message to the Discord channel, chunking if necessary to respect the 2000 character limit.
func (c *DiscordConnector) Send(channelID string, message string) error {
	// Discord message length limit: 2000 characters
	// Claude max_tokens: 1024 ≈ ~4000 chars worst case — must chunk
	chunks := chunkMessage(message, 1900) // 1900 to leave room for formatting
	for _, chunk := range chunks {
		_, err := c.session.ChannelMessageSend(channelID, chunk)
		if err != nil {
			// Check for rate limit (HTTP 429)
			if restErr, ok := err.(*discordgo.RESTError); ok {
				if restErr.Response.StatusCode == 429 {
					// discordgo handles rate limits internally — this shouldn't happen
					// but if it does, log and continue
					log.Printf("discord rate limited on channel %s", channelID)
					time.Sleep(1 * time.Second)
					_, err = c.session.ChannelMessageSend(channelID, chunk)
				}
			}
			if err != nil {
				return fmt.Errorf("discord send to %s: %w", channelID, err)
			}
		}
	}
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
		// Walk back to find a space
		for split > 0 && msg[split] != ' ' {
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
