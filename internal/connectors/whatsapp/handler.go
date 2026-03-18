package whatsapp

import (
	"context"
	"log"
	"strings"

	"go.mau.fi/whatsmeow/types/events"

	"bruce/internal/worker"
)

// handleEvent is the whatsmeow event handler registered in New().
func (c *WhatsAppConnector) handleEvent(evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		c.handleMessage(e)
	case *events.Disconnected:
		log.Println("INFO: whatsapp: disconnected from server")
	case *events.LoggedOut:
		log.Printf("INFO: whatsapp: logged out (reason: %v) — deleting device store", e.Reason)
		if err := c.client.Store.Delete(context.Background()); err != nil {
			log.Printf("ERROR: whatsapp: failed to delete device store: %v", err)
		}
	}
}

// handleMessage processes an incoming WhatsApp message event.
func (c *WhatsAppConnector) handleMessage(msg *events.Message) {
	// Ignore self-sent messages.
	if msg.Info.IsFromMe {
		return
	}
	// Phase 1: DMs only — ignore group messages.
	if msg.Info.IsGroup {
		return
	}

	// Extract plain text.
	var text string
	if msg.Message != nil {
		if msg.Message.GetConversation() != "" {
			text = msg.Message.GetConversation()
		} else if msg.Message.GetExtendedTextMessage() != nil {
			text = msg.Message.GetExtendedTextMessage().GetText()
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	channelID := msg.Info.Sender.User // raw phone number, no JID suffix
	log.Printf("DEBUG: whatsapp: incoming message from %s, len=%d", channelID, len(text))

	payload := worker.ProcessIncomingMessagePayload{
		ConnectorType: "whatsapp",
		ChannelID:     channelID,
		Content:       text,
	}
	task, err := worker.NewProcessIncomingMessageTask(payload)
	if err != nil {
		log.Printf("ERROR: whatsapp: create task: %v", err)
		return
	}
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		log.Printf("ERROR: whatsapp: enqueue task for %s: %v", channelID, err)
		return
	}
	log.Printf("DEBUG: whatsapp: task enqueued for channel %s", channelID)
}
