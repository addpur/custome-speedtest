package bundlefx

import (
	"context"
	"log"

	"github.com/addpur/custome-speedtest/bundlefx/entity"
	"github.com/addpur/custome-speedtest/common"
	"github.com/addpur/custome-speedtest/fe"
	"github.com/addpur/custome-speedtest/rest"
	"github.com/addpur/custome-speedtest/rest/delivery"
	"github.com/addpur/custome-speedtest/speedtest"
	"github.com/addpur/custome-speedtest/speedtest/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"
)

// NewFiberApp initializes Fiber HTTP engine with CORS and recovery middleware
func NewFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		AppName:               "Custome Speedtest v1.0.0",
		BodyLimit:             100 * 1024 * 1024, // 100MB upload limit
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Cache-Control, Pragma",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	app.Use(recover.New())
	return app
}

// RegisterUnifiedLifecycle unifies all startup and shutdown lifecycles:
// - Route registration (REST & Static Frontend)
// - Scheduler background tasks
// - Fiber HTTP server listener & graceful shutdown
func RegisterUnifiedLifecycle(
	lc fx.Lifecycle,
	app *fiber.App,
	handler *delivery.Handler,
	scheduler domain.Scheduler,
	cfg domain.Config,
) {
	// Register REST API endpoints
	handler.RegisterRoutes(app)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start background scheduler if configured
			scheduler.Start(ctx)

			// Start Fiber HTTP server
			addr := cfg.ListenAddr
			if addr == "" {
				addr = common.DefaultListenAddr
			}
			go func() {
				log.Printf("Custome Speedtest server listening on %s", addr)
				if err := app.Listen(addr); err != nil {
					log.Printf("Server stopped: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Shutting down Custome Speedtest server...")
			return app.ShutdownWithContext(ctx)
		},
	})
}

var CoreModules = fx.Module("core",
	fx.Provide(NewFiberApp),
	speedtest.Module,
	rest.Module,
	fe.Module,
	fx.Invoke(RegisterUnifiedLifecycle),
)

var EntityModules = fx.Module("entity",
	entity.Module,
)
