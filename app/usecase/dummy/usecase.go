package usecase

import (
	"bruce/app/entities"
)

type DummyRepository interface {
	Create(dummy entities.Dummy) error
	GetDummyByID(id int64) (entities.Dummy, error)
	UpdateDummy(dummy entities.Dummy) error
	GetAllDummies() ([]entities.Dummy, error)
}

type DummyService interface {
	ProcessDummy(dummy entities.Dummy) (entities.Dummy, error)
}

type DummyUseCase struct {
	Repo    DummyRepository
	Service DummyService
}

func NewDummyUseCase(r DummyRepository, s DummyService) *DummyUseCase {
	return &DummyUseCase{
		Repo:    r,
		Service: s,
	}
}

func (u *DummyUseCase) CreateDummy(text string) error {
	d := entities.Dummy{Text: text}
	return u.Repo.Create(d)
}

func (u *DummyUseCase) GetDummy(id int64) (entities.Dummy, error) {
	return u.Repo.GetDummyByID(id)
}

func (u *DummyUseCase) ProcessDummy(id int64) error {
	d, err := u.Repo.GetDummyByID(id)
	if err != nil {
		return err
	}
	
	d, err = u.Service.ProcessDummy(d)
	if err != nil {
		return err
	}
	
	return u.Repo.UpdateDummy(d)
}

func (u *DummyUseCase) GetAllDummies() ([]entities.Dummy, error) {
	return u.Repo.GetAllDummies()
}
