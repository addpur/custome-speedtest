package entity

import (
    "go.uber.org/fx"
)

var Module = fx.Module("entity",
    fx.Provide(NewConfig),
    fx.Invoke(RegisterSignals),
)
