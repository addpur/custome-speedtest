package fe

import (
	"embed"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

//go:embed static/*
var StaticFS embed.FS

var Module = fx.Module("fe",
	fx.Provide(func() http.FileSystem {
		return http.FS(StaticFS)
	}),
	fx.Invoke(func(app *fiber.App) {
		// Serve root as index.html
		app.Get("/", func(c *fiber.Ctx) error {
			data, err := StaticFS.ReadFile("static/index.html")
			if err != nil {
				return c.Status(404).SendString("index.html not found")
			}
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.Send(data)
		})

		// Serve static assets
		app.Get("/static/*", func(c *fiber.Ctx) error {
			path := c.Params("*")
			if path == "" || path == "/" {
				path = "index.html"
			}
			data, err := StaticFS.ReadFile("static/" + path)
			if err != nil {
				return c.Status(404).SendString("Not found")
			}

			switch {
			case strings.HasSuffix(path, ".html"):
				c.Set("Content-Type", "text/html; charset=utf-8")
			case strings.HasSuffix(path, ".css"):
				c.Set("Content-Type", "text/css; charset=utf-8")
			case strings.HasSuffix(path, ".js"):
				c.Set("Content-Type", "application/javascript; charset=utf-8")
			case strings.HasSuffix(path, ".svg"):
				c.Set("Content-Type", "image/svg+xml")
			case strings.HasSuffix(path, ".json"):
				c.Set("Content-Type", "application/json")
			case strings.HasSuffix(path, ".png"):
				c.Set("Content-Type", "image/png")
			case strings.HasSuffix(path, ".ico"):
				c.Set("Content-Type", "image/x-icon")
			}
			return c.Send(data)
		})
	}),
)

