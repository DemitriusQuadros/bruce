package handler

import (
	"context"
	"encoding/json"
	"bruce/app/entities"
	"log"

	"github.com/hibiken/asynq"
)

type DummyUseCase interface {
	ProcessDummy(id int64) error
}

type DummyProcessor struct {
	useCase DummyUseCase
}

func NewDummyProcessor(uc DummyUseCase) *DummyProcessor {
	return &DummyProcessor{
		useCase: uc,
	}
}

func (p *DummyProcessor) HandleDummyTask(ctx context.Context, t *asynq.Task) error {
	var dummy entities.Dummy

	if err := json.Unmarshal(t.Payload(), &dummy); err != nil {
		return err
	}

	err := p.useCase.ProcessDummy(dummy.ID)
	if err != nil {
		log.Printf("Error processing dummy task for ID %d: %v", dummy.ID, err)
		return err
	}

	log.Printf("Dummy task for ID %d processed successfully", dummy.ID)
	return nil
}
