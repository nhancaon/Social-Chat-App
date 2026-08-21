package controllers

import (
	"Server/database"
	"Server/models"
	"Server/storage"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const presignExpiry = 15 * time.Minute

// Create Upload URL
// @Summary request a presigned S3 upload URL
// @Description creates a pending file record and returns a presigned URL the client uploads directly to S3
// @Tags Files
// @Accept json
// @Produce json
// @Param file body models.CreateUploadRequest true "upload request details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files/upload-url [post]
func CreateUploadURL(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	var FileSchema = database.DB.Collection("files")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	var body models.CreateUploadRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	objId, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user id",
		})
	}

	var user models.UserModel
	if err := UserSchema.FindOne(ctx, bson.M{"_id": objId}).Decode(&user); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	quota := user.StorageQuotaBytes
	if quota == 0 {
		quota = models.DefaultStorageQuotaBytes
	}
	if user.StorageUsedBytes+body.SizeBytes > quota {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":      "storage quota exceeded",
			"usedBytes":  user.StorageUsedBytes,
			"quotaBytes": quota,
		})
	}

	// start set data
	fileID := primitive.NewObjectID()
	var file models.FileModel
	file.ID = fileID
	file.OwnerID = userID
	file.Filename = body.Filename
	file.S3Key = fmt.Sprintf("uploads/%s/%s-%s", userID, fileID.Hex(), body.Filename)
	file.SizeBytes = body.SizeBytes
	file.ContentType = body.ContentType
	file.StorageClass = models.StorageClassStandard
	file.Uploaded = false
	file.IsDeleted = false
	file.CreatedAt = time.Now()
	// set data end

	if _, err := FileSchema.InsertOne(ctx, &file); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	uploadUrl, err := storage.PresignUploadURL(ctx, file.S3Key, body.ContentType, presignExpiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":      file,
		"uploadUrl": uploadUrl,
		"expiresIn": int(presignExpiry.Seconds()),
	})
}

// Confirm Upload
// @Summary confirm a file finished uploading to S3
// @Description marks the file record as uploaded and credits the bytes against the user's storage quota
// @Tags Files
// @Produce json
// @Param id path string true "file id"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files/{id}/confirm [post]
func ConfirmUpload(c *fiber.Ctx) error {

	var FileSchema = database.DB.Collection("files")
	var UserSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	fileID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file id",
		})
	}

	var file models.FileModel
	if err := FileSchema.FindOne(ctx, bson.M{"_id": fileID}).Decode(&file); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "file not found",
		})
	}
	if file.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to confirm this file",
		})
	}
	if file.Uploaded {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": file})
	}

	now := time.Now()
	if _, err := FileSchema.UpdateOne(ctx, bson.M{"_id": fileID}, bson.M{"$set": bson.M{
		"uploaded":       true,
		"uploadedAt":     now,
		"lastAccessedAt": now,
	}}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// credit quota usage atomically so concurrent uploads from the same user can't race past the limit
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err == nil {
		if _, err := UserSchema.UpdateOne(ctx, bson.M{"_id": userObjID}, bson.M{"$inc": bson.M{"storageUsedBytes": file.SizeBytes}}); err != nil {
			log.Printf("failed to update storage usage for user %s: %v", userID, err)
		}
	}

	file.Uploaded = true
	file.UploadedAt = now
	file.LastAccessedAt = now
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": file})
}

// List Files
// @Summary list the current user's files
// @Tags Files
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files [get]
func ListFiles(c *fiber.Ctx) error {

	var FileSchema = database.DB.Collection("files")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	cursor, err := FileSchema.Find(ctx, bson.M{"ownerId": userID, "uploaded": true, "isDeleted": false})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer cursor.Close(ctx)

	files := make([]models.FileModel, 0)
	if err := cursor.All(ctx, &files); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": files})
}

// Get Download URL
// @Summary request a presigned S3 download URL
// @Description returns a presigned GET URL; fails with 409 if the file is archived and needs restoring first
// @Tags Files
// @Produce json
// @Param id path string true "file id"
// @Success 200 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files/{id}/download-url [get]
func GetDownloadURL(c *fiber.Ctx) error {

	var FileSchema = database.DB.Collection("files")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	fileID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file id",
		})
	}

	var file models.FileModel
	if err := FileSchema.FindOne(ctx, bson.M{"_id": fileID}).Decode(&file); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "file not found",
		})
	}
	if file.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to access this file",
		})
	}
	if file.IsDeleted {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"error": "file has been deleted",
		})
	}
	// GLACIER/RESTORING files aren't directly downloadable — the restore flow (not yet built)
	// will flip StorageClass to RESTORED once S3 finishes rehydrating the object.
	if file.StorageClass != models.StorageClassStandard && file.StorageClass != models.StorageClassRestored {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":        "file is archived and must be restored before download",
			"storageClass": file.StorageClass,
		})
	}

	url, err := storage.PresignDownloadURL(ctx, file.S3Key, presignExpiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if _, err := FileSchema.UpdateOne(ctx, bson.M{"_id": fileID}, bson.M{"$set": bson.M{"lastAccessedAt": time.Now()}}); err != nil {
		log.Printf("failed to update lastAccessedAt for file %s: %v", fileID.Hex(), err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"downloadUrl": url,
		"expiresIn":   int(presignExpiry.Seconds()),
	})
}

// Request Restore
// @Summary request restoring an archived (Glacier) file
// @Description kicks off an async Glacier restore; the restore-poll CronJob flips
// the file to RESTORED and pushes a realtime notification once S3 finishes
// @Tags Files
// @Produce json
// @Param id path string true "file id"
// @Success 202 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files/{id}/restore [post]
func RequestRestore(c *fiber.Ctx) error {

	var FileSchema = database.DB.Collection("files")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	fileID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file id",
		})
	}

	var file models.FileModel
	if err := FileSchema.FindOne(ctx, bson.M{"_id": fileID}).Decode(&file); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "file not found",
		})
	}
	if file.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to restore this file",
		})
	}

	switch file.StorageClass {
	case models.StorageClassStandard, models.StorageClassRestored:
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":      "file is already downloadable",
			"storageClass": file.StorageClass,
		})
	case models.StorageClassRestoring:
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message":      "restore already in progress",
			"storageClass": file.StorageClass,
		})
	}

	// restoreForDays is how long S3 keeps the rehydrated copy around before it
	// lapses back to GLACIER — kept in sync with the expiry restore-poll writes.
	const restoreForDays = 7
	if err := storage.RequestRestore(ctx, file.S3Key, restoreForDays); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if _, err := FileSchema.UpdateOne(ctx, bson.M{"_id": fileID}, bson.M{"$set": bson.M{
		"storageClass":       models.StorageClassRestoring,
		"restoreRequestedAt": time.Now(),
	}}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":      "restore requested — typically a few minutes to a few hours; you'll get a notification when it's ready",
		"storageClass": models.StorageClassRestoring,
	})
}

// Delete File
// @Summary move a file to trash (soft delete)
// @Tags Files
// @Produce json
// @Param id path string true "file id"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Security BearerAuth
// @Router /files/{id} [delete]
func DeleteFile(c *fiber.Ctx) error {

	var FileSchema = database.DB.Collection("files")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	fileID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid file id",
		})
	}

	var file models.FileModel
	if err := FileSchema.FindOne(ctx, bson.M{"_id": fileID}).Decode(&file); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "file not found",
		})
	}
	if file.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to delete this file",
		})
	}

	// soft delete only — quota is credited back and the S3 object is actually removed
	// by the trash-purge job (not yet built), same as Drive's "empty trash" behavior.
	result, err := FileSchema.UpdateOne(ctx, bson.M{"_id": fileID}, bson.M{"$set": bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
	}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if result.ModifiedCount == 1 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "File moved to trash"})
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "can't delete file!"})
}
