package routes

import (
	"Server/controllers"
	"Server/middleware"
	"Server/validation"

	"github.com/gofiber/fiber/v2"
)

func SetupFileRoutes(app *fiber.App) {
	// request a presigned URL, client uploads straight to S3 (backend never proxies file bytes)
	app.Post("/files/upload-url", middleware.AuthMiddleware, validation.ValidateUploadRequest, controllers.CreateUploadURL)
	// client calls this after the S3 upload succeeds, to credit quota + mark the file ready
	app.Post("/files/:id/confirm", middleware.AuthMiddleware, controllers.ConfirmUpload)
	app.Get("/files", middleware.AuthMiddleware, controllers.ListFiles)
	app.Get("/files/:id/download-url", middleware.AuthMiddleware, controllers.GetDownloadURL)
	// archived (GLACIER) files need this before they're downloadable again
	app.Post("/files/:id/restore", middleware.AuthMiddleware, controllers.RequestRestore)
	// soft delete (trash) — permanent delete happens later via the trash-purge job
	app.Delete("/files/:id", middleware.AuthMiddleware, controllers.DeleteFile)
}
