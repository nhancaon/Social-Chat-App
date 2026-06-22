package main

import (
	"Server/database"
	_ "Server/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

// @title Fiber Golang Rest API
// @version 1.0
// @description this is  Swagger docs for Social chat API created with golang and fiber
// @host localhost:5000
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	database.Connect()
	app := fiber.New()

	app.Use(cors.New(
		cors.Config{
			// AllowOrigins:     "*",
			// AllowHeaders:     "Content-Type, Authorization",
			// AllowMethods:     "GET, POST, PUT, DELETE",
			AllowCredentials: true,
			AllowOriginsFunc: func(origin string) bool {
				return true
			},
		},
	))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	//Server swagger docs
	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Listen(":5000")
}
