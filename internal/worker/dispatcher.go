package worker

import (
	"fmt"
	"log"
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
	log.Printf("DEBUG: dispatcher registered for connector type: %s", connectorType)
}

// Dispatch forwards a message to the registered Dispatcher for connectorType.
func (r *DispatcherRegistry) Dispatch(connectorType, channelID, message string) error {
	r.mu.RLock()
	d, ok := r.dispatchers[connectorType]
	registeredTypes := make([]string, 0, len(r.dispatchers))
	for ct := range r.dispatchers {
		registeredTypes = append(registeredTypes, ct)
	}
	r.mu.RUnlock()

	if !ok {
		log.Printf("ERROR: dispatcher not found for connector: %s (registered: %v)", connectorType, registeredTypes)
		return fmt.Errorf("no dispatcher registered for connector: %s", connectorType)
	}

	log.Printf("DEBUG: dispatching to %s dispatcher - channel=%s, message_len=%d", connectorType, channelID, len(message))
	if err := d.Send(channelID, message); err != nil {
		log.Printf("ERROR: dispatcher Send failed - connector=%s, channel=%s, error=%v", connectorType, channelID, err)
		return err
	}
	log.Printf("DEBUG: dispatcher Send succeeded - connector=%s, channel=%s", connectorType, channelID)
	return nil
}
