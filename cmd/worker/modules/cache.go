package modules

import (
	"bruce/internal/memcache"

	"go.uber.org/fx"
)

var CacheModule = fx.Module("cache",
	fx.Provide(memcache.NewInMemoryCache),
)
