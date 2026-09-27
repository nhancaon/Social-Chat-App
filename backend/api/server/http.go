package server

import (
	_ "Server/docs"
	"Server/metrics"
	"Server/routes"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewHTTPServer() *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(metrics.HTTPMiddleware())

	app.Use(cors.New(
		cors.Config{
			AllowCredentials: true,
			AllowOriginsFunc: func(origin string) bool {
				return true
			},
		},
	))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to Social app")
	})

	routes.SetupAuthRoutes(app)
	routes.SetupUserRoutes(app)
	routes.SetupPostRoutes(app)
	routes.SetupChatRoutes(app)
	routes.SetupNotificationRoutes(app)
	routes.SetupFileRoutes(app)

	app.Get("/swagger/*", swagger.HandlerDefault)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	return app
}
