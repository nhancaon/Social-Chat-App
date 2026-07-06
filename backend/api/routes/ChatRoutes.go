package routes

import (
	"Server/controllers"
	"Server/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupChatRoutes(app *fiber.App) {
	chat := app.Group("/chat", middleware.AuthMiddleware)
	chat.Post("/sendmessage", controllers.SendMessage)
	chat.Get("/getmsgsbynums", controllers.GetMsgsByNums)
	chat.Get("/get-user-unreadmsg", controllers.GetUserUnreadMsg)
	chat.Get("/mark-msg-asreaded", controllers.MarkMsgAsReaded)

}
