package routes

import (
	"Server/controllers"
	"Server/middleware"
	"Server/realtime"

	"github.com/gofiber/fiber/v2"
)

func SetupNotificationRoutes(app *fiber.App) {
	app.Get("/notification/mark-notification-asreaded", middleware.AuthMiddleware, controllers.MarkNotificationAsRead)

	// Registered before the /notification/:userid wildcard below — otherwise
	// that route would match "ws" as the :userid param and shadow this one.
	app.Get("/notification/ws", middleware.WSAuthMiddleware, func(c *fiber.Ctx) error {
		notificationHub := realtime.GetNotificationHub()
		if notificationHub == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "notification service not ready")
		}
		return notificationHub.HandleClient(c)
	})

	app.Get("/notification/:userid", middleware.AuthMiddleware, controllers.GetUserNotification)
}
