package main

import (
	"Server/database"
	_ "Server/docs"
	"Server/routes"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
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

	//load env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading env", err)
	}

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

	//Setup routes
	routes.SetupAuthRoutes(app)
	routes.SetupUserRoutes(app)
	routes.SetupPostRoutes(app)
	routes.SetupChatRoutes(app)
	routes.SetupNotificationRoutes(app)

	//Server swagger docs
	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Listen(":5000")
}
