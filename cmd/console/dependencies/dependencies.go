package dependencies

import (
	"bruce/internal/configuration"
	"bruce/internal/db"

	"gorm.io/gorm"
)

type Dependencies struct {
	Cfg *configuration.Configuration
	Db  *gorm.DB
}

func Init() *Dependencies {
	cfg := configuration.NewConfiguration()
	db, err := db.NewDatabase(cfg)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	return &Dependencies{
		Cfg: cfg,
		Db:  db,
	}
}
