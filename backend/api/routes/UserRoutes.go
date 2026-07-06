package routes

import (
	"Server/controllers"
	"Server/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app *fiber.App) {
	app.Get("/user/getUser/:id", controllers.GetUserByID)
	app.Get("/user/getSug", controllers.GetSugUser)
	app.Post("/user/batch", middleware.AuthMiddleware, controllers.GetUsersByIDs)
	app.Patch("/user/Update/:id", middleware.AuthMiddleware, controllers.UpdateUser)
	app.Patch("/user/:id/following", middleware.AuthMiddleware, controllers.FollowingUser)
	app.Delete("/user/delete/:id", middleware.AuthMiddleware, controllers.DeleteUser)

}
