package modules

import (
	"bruce/internal/db"

	"go.uber.org/fx"
)

var DbModule = fx.Module("db",
	fx.Provide(
		db.NewDatabase,
	),
)
