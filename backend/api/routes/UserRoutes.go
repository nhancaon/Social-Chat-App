package routes

import (
	"Server/controllers"
	"Server/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app *fiber.App) {
	// auth
	app.Get("/user/getUser/:id", controllers.GetUserByID)
	// getSug
	app.Get("/user/getSug", controllers.GetSugUser)
	// Update
	app.Patch("/user/Update/:id", middleware.AuthMiddleware, controllers.UpdateUser)
	// following
	app.Patch("/user/:id/following", middleware.AuthMiddleware, controllers.FollowingUser)
	// delete
	app.Delete("/user/delete/:id", middleware.AuthMiddleware, controllers.DeleteUser)

}
