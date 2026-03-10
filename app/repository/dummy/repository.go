package repository

import (
	"bruce/app/entities"

	"gorm.io/gorm"
)

type DummyRepository struct {
	db *gorm.DB
}

func NewDummyRepository(db *gorm.DB) DummyRepository {
	return DummyRepository{
		db: db,
	}
}

func (r DummyRepository) Create(dummy entities.Dummy) error {
	return r.db.Create(&dummy).Error
}

func (r DummyRepository) GetDummyByID(id int64) (entities.Dummy, error) {
	var dummy entities.Dummy
	err := r.db.Where("id = ?", id).First(&dummy).Error
	if err != nil {
		return entities.Dummy{}, err
	}
	return dummy, nil
}

func (r DummyRepository) UpdateDummy(dummy entities.Dummy) error {
	return r.db.Save(&dummy).Error
}

func (r DummyRepository) GetAllDummies() ([]entities.Dummy, error) {
	var dummies []entities.Dummy
	err := r.db.Find(&dummies).Error
	return dummies, err
}
