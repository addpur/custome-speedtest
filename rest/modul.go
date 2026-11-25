package rest

import (
	"github.com/addpur/custome-speedtest/rest/delivery"
	"github.com/addpur/custome-speedtest/rest/service"
	"go.uber.org/fx"
)

var Module = fx.Module("rest",
	fx.Provide(
		service.NewAPIService,
		delivery.NewHandler,
	),
)
