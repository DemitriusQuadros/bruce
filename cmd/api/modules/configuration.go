package modules

import (
	"bruce/internal/configuration"

	"go.uber.org/fx"
)

var ConfigurationModule = fx.Module("configuration",
	fx.Provide(configuration.NewConfiguration),
)
