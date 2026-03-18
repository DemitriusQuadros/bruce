// Package whatsapp provides a WhatsApp connector for Bruce using whatsmeow.
package whatsapp

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	"bruce/internal/config"
)

// WhatsAppConnector connects to WhatsApp via whatsmeow and implements worker.Dispatcher.
type WhatsAppConnector struct {
	client      *whatsmeow.Client
	asynqClient *asynq.Client
	container   *sqlstore.Container
}

// New creates and initialises the WhatsApp connector (does not connect yet).
func New(cfg *config.Config, asynqClient *asynq.Client) (*WhatsAppConnector, error) {
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", cfg.Connectors.WhatsApp.DeviceStoreDSN, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: open device store: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: get device: %w", err)
	}

	client := whatsmeow.NewClient(device, waLog.Noop)

	c := &WhatsAppConnector{
		client:      client,
		asynqClient: asynqClient,
		container:   container,
	}
	client.AddEventHandler(c.handleEvent)
	return c, nil
}

// Connect authenticates with WhatsApp (QR code if not yet paired, reconnect otherwise).
func (c *WhatsAppConnector) Connect() error {
	return c.connect()
}

// Disconnect closes the WhatsApp WebSocket connection gracefully.
func (c *WhatsAppConnector) Disconnect() {
	c.client.Disconnect()
}

// Send implements worker.Dispatcher. It sends a text message to the given phone number.
// channelID is the raw phone number (e.g. "15551234567"), without the JID suffix.
func (c *WhatsAppConnector) Send(channelID, message string) error {
	jid := types.NewJID(channelID, types.DefaultUserServer)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := c.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &message,
	})
	if err != nil {
		return fmt.Errorf("whatsapp send to %s: %w", channelID, err)
	}
	log.Printf("DEBUG: whatsapp sent message to %s", channelID)
	return nil
}
