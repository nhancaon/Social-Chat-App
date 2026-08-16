package routes

import (
	"Server/controllers"
	"Server/middleware"
	"Server/realtime"

	"github.com/gofiber/fiber/v2"
)

func SetupChatRoutes(app *fiber.App) {
	// NOTE: not using app.Group("/chat", middleware.AuthMiddleware) here —
	// Fiber registers a group's middleware as a prefix-matching "Use" handler,
	// so it would also intercept /chat/ws below (which needs WSAuthMiddleware,
	// not AuthMiddleware) since that path also starts with "/chat".
	app.Post("/chat/sendmessage", middleware.AuthMiddleware, controllers.SendMessage)
	app.Get("/chat/getmsgsbynums", middleware.AuthMiddleware, controllers.GetMessagesByPage)
	app.Get("/chat/get-user-unreadmsg", middleware.AuthMiddleware, controllers.GetUserUnreadMsg)
	app.Get("/chat/mark-msg-asreaded", middleware.AuthMiddleware, controllers.MarkMessageAsRead)

	app.Get("/chat/ws", middleware.WSAuthMiddleware, func(c *fiber.Ctx) error {
		chatHub := realtime.GetChatHub()
		if chatHub == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "chat service not ready")
		}
		return chatHub.HandleClient(c)
	})
}
