package handler_test

import (
	"context"
	"encoding/json"
	"bruce/app/entities"
	handler "bruce/app/handler/tasks/dummy"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDummyUseCase struct {
	mock.Mock
}

func (m *MockDummyUseCase) ProcessDummy(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestDummyProcessor_HandleDummyTask(t *testing.T) {
	uc := new(MockDummyUseCase)
	processor := handler.NewDummyProcessor(uc)

	dummy := entities.Dummy{ID: 1, Text: "Test"}
	payload, _ := json.Marshal(dummy)
	task := asynq.NewTask("dummy:task:process", payload)

	uc.On("ProcessDummy", int64(1)).Return(nil)

	err := processor.HandleDummyTask(context.Background(), task)
	assert.NoError(t, err)
	uc.AssertExpectations(t)
}
