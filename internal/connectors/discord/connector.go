// Package discord provides a Discord connector stub for Bruce.
package discord

import (
	"context"
	"log"
)

// Connector is the interface for a Discord connector.
type Connector interface {
	Start(ctx context.Context) error
	Stop() error
}

// DisabledConnector is a no-op connector returned when Discord is disabled in config.
type DisabledConnector struct{}

// NewConnector returns a DisabledConnector stub.
// The real implementation will be added in a later spec.
func NewConnector() Connector {
	return &DisabledConnector{}
}

// Start logs that Discord is disabled and returns nil.
func (c *DisabledConnector) Start(ctx context.Context) error {
	log.Println("INFO: Discord connector is disabled — skipping start")
	return nil
}

// Stop is a no-op for the disabled connector.
func (c *DisabledConnector) Stop() error {
	return nil
}
