package worker

import (
	"fmt"
	"sync"
)

// Dispatcher sends a response back to a specific channel on a connector.
type Dispatcher interface {
	Send(channelID string, message string) error
}

// DispatcherRegistry maps connector type strings to their Dispatcher implementations.
// Registered at startup in main.go before the worker starts.
type DispatcherRegistry struct {
	dispatchers map[string]Dispatcher
	mu          sync.RWMutex
}

// NewDispatcherRegistry returns an empty DispatcherRegistry ready for registration.
func NewDispatcherRegistry() *DispatcherRegistry {
	return &DispatcherRegistry{dispatchers: make(map[string]Dispatcher)}
}

// Register binds a Dispatcher to a connector type name (e.g. "whatsapp", "discord").
func (r *DispatcherRegistry) Register(connectorType string, d Dispatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dispatchers[connectorType] = d
}

// Dispatch forwards a message to the registered Dispatcher for connectorType.
func (r *DispatcherRegistry) Dispatch(connectorType, channelID, message string) error {
	r.mu.RLock()
	d, ok := r.dispatchers[connectorType]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no dispatcher registered for connector: %s", connectorType)
	}
	return d.Send(channelID, message)
}
