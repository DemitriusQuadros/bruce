package workers

import (
	"encoding/json"
	"bruce/app/entities"
	"bruce/internal/configuration"
	"log"

	"github.com/hibiken/asynq"
)

type DummyWorker struct {
	client *asynq.Client
}

func NewDummyWorker(cfg *configuration.Configuration) DummyWorker {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Addr})

	return DummyWorker{
		client: client,
	}
}

const (
	DummyTask = "dummy:task:process"
)

func (w DummyWorker) EnqueueDummyTask(dummy entities.Dummy) error {
	payload, err := json.Marshal(dummy)
	if err != nil {
		return err
	}
	
	t := asynq.NewTask(DummyTask, payload)
	info, err := w.client.Enqueue(t)
	if err != nil {
		return err
	}
	
	log.Printf(" [*] Successfully enqueued dummy task: %+v", info.ID)
	return nil
}
