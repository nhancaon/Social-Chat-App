package controllers

import (
	"Server/database"
	"Server/models"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MarkNotAsReaded
// @Summary Mark notifications as read for a user
// @Description MarkNotAsReaded
// @Tags Notifications
// @Accept json
// @Produce json
// @Param id query string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /notification/mark-notification-asreaded [get]
func MarknotAsReaded(c *fiber.Ctx) error {

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	id := c.Query("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id in query is required",
		})
	}

	// a user can only mark their own notifications as read
	if id != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you are not allowed to modify another user's notifications",
		})
	}

	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// exact match on mainuid, never use $regex for an ID lookup:
	// regex does substring matching (not equality) and is vulnerable to
	// regex injection if the value isn't escaped
	filter := bson.M{"mainuid": id}
	update := bson.M{"$set": bson.M{"isreded": true}}

	result, err := NotificationSchema.UpdateMany(ctx, filter, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to mark notifications as read",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":       true,
		"modifiedCount": result.ModifiedCount,
	})
}

// GetUserNotification
// @Summary Get user notifications
// @Description GetUserNotification
// @Tags Notifications
// @Accept json
// @Produce json
// @Param userid path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /notification/{userid} [get]
func GetUserNotification(c *fiber.Ctx) error {

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	id := c.Params("userid")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "userid in params is required",
		})
	}

	// a user can only read their own notifications
	if id != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you are not allowed to view another user's notifications",
		})
	}

	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// exact match on mainuid (see comment in MarknotAsReaded for why
	// this must never be a $regex match)
	filter := bson.M{"mainuid": id}
	findOptions := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := NotificationSchema.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to retrieve notifications",
			"error":   err.Error(),
		})
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to decode notifications",
			"error":   err.Error(),
		})
	}

	if notifications == nil {
		notifications = []models.Notification{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"notifications": notifications,
	})
}
