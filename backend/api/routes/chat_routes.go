package routes

import (
	"Server/controllers"
	"Server/middleware"
	"Server/realtime"

	"github.com/gofiber/fiber/v2"
)

func SetupChatRoutes(app *fiber.App) {
	chat := app.Group("/chat", middleware.AuthMiddleware)
	chat.Post("/sendmessage", controllers.SendMessage)
	chat.Get("/getmsgsbynums", controllers.GetMessagesByPage)
	chat.Get("/get-user-unreadmsg", controllers.GetUserUnreadMsg)
	chat.Get("/mark-msg-asreaded", controllers.MarkMessageAsRead)

	app.Get("/chat/ws", middleware.WSAuthMiddleware, func(c *fiber.Ctx) error {
		hub := realtime.GetChatHub()
		if hub == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "chat service not ready")
		}
		return hub.HandleClient(c)
	})
}
