package speedtest

import (
	"github.com/addpur/custome-speedtest/speedtest/service"
	"go.uber.org/fx"
)

var Module = fx.Module("speedtest",
	fx.Provide(
		service.NewEngine,
		service.NewRegistry,
		service.NewStore,
		service.NewScheduler,
	),
)
