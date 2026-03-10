package services

import (
	"fmt"
	"bruce/app/entities"
)

// DummyService is an interface demonstrating how external services or complex
// domain logic should be structured in this scaffolding.
type DummyService interface {
	ProcessDummy(dummy entities.Dummy) (entities.Dummy, error)
}

type dummyService struct{}

func NewDummyService() DummyService {
	return &dummyService{}
}

// ProcessDummy demonstrates some business logic or external call.
func (s *dummyService) ProcessDummy(dummy entities.Dummy) (entities.Dummy, error) {
	// Example: just modify the text
	dummy.Text = fmt.Sprintf("Processed: %s", dummy.Text)
	return dummy, nil
}
