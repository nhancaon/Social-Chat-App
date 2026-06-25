package routes

import (
	"Server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupNotificationRoutes(app *fiber.App) {
	// auth
	app.Get("/notification/mark-notification-asreaded", controllers.MarknotAsReaded)
	app.Get("/notification/:userid", controllers.GetUserNotification)

}
