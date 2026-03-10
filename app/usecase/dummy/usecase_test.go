package usecase_test

import (
	"bruce/app/entities"
	usecase "bruce/app/usecase/dummy"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDummyRepository struct {
	mock.Mock
}

func (m *MockDummyRepository) Create(dummy entities.Dummy) error {
	args := m.Called(dummy)
	return args.Error(0)
}

func (m *MockDummyRepository) GetDummyByID(id int64) (entities.Dummy, error) {
	args := m.Called(id)
	return args.Get(0).(entities.Dummy), args.Error(1)
}

func (m *MockDummyRepository) GetAllDummies() ([]entities.Dummy, error) {
	args := m.Called()
	return args.Get(0).([]entities.Dummy), args.Error(1)
}

func (m *MockDummyRepository) UpdateDummy(dummy entities.Dummy) error {
	args := m.Called(dummy)
	return args.Error(0)
}

type MockDummyService struct {
	mock.Mock
}

func (m *MockDummyService) ProcessDummy(dummy entities.Dummy) (entities.Dummy, error) {
	args := m.Called(dummy)
	return args.Get(0).(entities.Dummy), args.Error(1)
}

func TestDummyUseCase_CreateDummy(t *testing.T) {
	repo := new(MockDummyRepository)
	svc := new(MockDummyService)
	uc := usecase.NewDummyUseCase(repo, svc)

	expectedDummy := entities.Dummy{Text: "Test"}
	repo.On("Create", expectedDummy).Return(nil)

	err := uc.CreateDummy("Test")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDummyUseCase_GetDummy(t *testing.T) {
	repo := new(MockDummyRepository)
	svc := new(MockDummyService)
	uc := usecase.NewDummyUseCase(repo, svc)

	expectedDummy := entities.Dummy{ID: 1, Text: "Test"}
	repo.On("GetDummyByID", int64(1)).Return(expectedDummy, nil)

	result, err := uc.GetDummy(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedDummy, result)
	repo.AssertExpectations(t)
}

func TestDummyUseCase_ProcessDummy(t *testing.T) {
	repo := new(MockDummyRepository)
	svc := new(MockDummyService)
	uc := usecase.NewDummyUseCase(repo, svc)

	initialDummy := entities.Dummy{ID: 1, Text: "Test"}
	processedDummy := entities.Dummy{ID: 1, Text: "Processed: Test"}

	repo.On("GetDummyByID", int64(1)).Return(initialDummy, nil)
	svc.On("ProcessDummy", initialDummy).Return(processedDummy, nil)
	repo.On("UpdateDummy", processedDummy).Return(nil)

	err := uc.ProcessDummy(1)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}
