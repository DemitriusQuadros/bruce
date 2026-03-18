package whatsapp

import (
	"context"
	"fmt"
	"log"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
)

// connect handles the WhatsApp authentication flow.
// If the device store already has credentials it reconnects directly;
// otherwise it opens a QR channel and waits for the user to scan.
func (c *WhatsAppConnector) connect() error {
	if c.client.Store.ID != nil {
		// Already paired — just reconnect.
		log.Println("INFO: whatsapp: reconnecting with stored credentials")
		return c.client.Connect()
	}

	// Not yet paired — open QR channel (must be called before Connect).
	ctx := context.Background()
	qrCh, err := c.client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: get QR channel: %w", err)
	}

	if err := c.client.Connect(); err != nil {
		return fmt.Errorf("whatsapp: connect for QR: %w", err)
	}

	log.Println("INFO: whatsapp: waiting for QR code scan — open WhatsApp on your phone and scan the code below")

	for item := range qrCh {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			printQRToTerminal(item.Code)
		case "success":
			log.Println("INFO: whatsapp: QR pairing successful")
			return nil
		case "timeout":
			return fmt.Errorf("whatsapp: QR code timed out — restart the server to try again")
		default:
			if item.Error != nil {
				return fmt.Errorf("whatsapp: QR pairing error (%s): %w", item.Event, item.Error)
			}
			return fmt.Errorf("whatsapp: unexpected QR channel event: %s", item.Event)
		}
	}
	return fmt.Errorf("whatsapp: QR channel closed unexpectedly")
}

// printQRToTerminal renders the QR code as Unicode half-blocks to stdout.
func printQRToTerminal(code string) {
	qrterminal.GenerateHalfBlock(code, qrterminal.L, log.Writer())
}
