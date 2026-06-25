package controllers

import (
	"Server/database"
	"Server/models"
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SendMessage
// @Summary send message to friend user
// @Description SendMessage from one user to another
// @Tags Chat
// @Accept json
// @Produce json
// @Param message body models.SendMessageM true "user SendMessage details"
// @Success 201 {object} models.Message
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /chat/sendmessage [post]
func SendMessage(c *fiber.Ctx) error {

	var MessageSchema = database.DB.Collection("messages")
	var UnReadMsgSchema = database.DB.Collection("unReadedmessages")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	var body models.SendMessageM
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// manual validation matching the `validate` tags on SendMessageM
	if len(body.Content) < 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "content is required and must be at least 5 characters",
		})
	}
	if body.Receiver == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "recever is required",
		})
	}
	if body.Receiver == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "you can't send a message to yourself",
		})
	}

	// map to the persisted Message; sender always comes from the auth token,
	// never trust body.Sender to avoid spoofing who sent the message
	msg := models.Message{
		Content:   body.Content,
		Sender:    userID,
		Receiver:  body.Receiver,
		CreatedAt: time.Now(),
	}

	// save the message to db
	result, err := MessageSchema.InsertOne(ctx, &msg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to save message",
			"details": err.Error(),
		})
	}

	// update or create the unread message count for the receiver
	filter := bson.M{"mainUserid": msg.Receiver, "otherUserid": msg.Sender}
	update := bson.M{"$inc": bson.M{"numOfUnreadMessages": 1}, "$set": bson.M{"isRead": false}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var unReadMsg models.UnReadMsg
	if err := UnReadMsgSchema.FindOneAndUpdate(ctx, filter, update, opts).Decode(&unReadMsg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to update unread message count",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Message sent successfully",
		"result":  result.InsertedID,
	})
}

// GetMsgsByNums
// @Summary get messages by pagination
// @Description Get messages between two users by pagination
// @Tags Chat
// @Accept json
// @Produce json
// @Param from query int true "Starting page num"
// @Param firstuid query string true "first user id"
// @Param seconduid query string true "second user id"
// @Success 200 {object} []models.Message
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /chat/getmsgsbynums [get]
func GetMsgsByNums(c *fiber.Ctx) error {

	var MessageSchema = database.DB.Collection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	from, err := strconv.Atoi(c.Query("from"))
	if err != nil || from < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "invalid value for from",
		})
	}

	firstuid := c.Query("firstuid")
	seconduid := c.Query("seconduid")
	if firstuid == "" || seconduid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "firstuid and seconduid query params are required",
		})
	}

	// the caller must be one of the two participants in the conversation
	if userID != firstuid && userID != seconduid {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you are not allowed to view this conversation",
		})
	}

	// construct the filter
	senderFilter := bson.M{"sender": firstuid, "recever": seconduid}
	receiverFilter := bson.M{"sender": seconduid, "recever": firstuid}
	filter := bson.M{"$or": []bson.M{senderFilter, receiverFilter}}

	const LIMIT = 2

	// pagination options: sort ascending so we don't need to reverse afterwards
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "_id", Value: -1}})
	findOptions.SetSkip(int64(from) * LIMIT)
	findOptions.SetLimit(LIMIT)

	cursor, err := MessageSchema.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to retrieve messages",
			"error":   err.Error(),
		})
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	for cursor.Next(ctx) {
		var msg models.Message
		if err := cursor.Decode(&msg); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "failed to decode messages",
				"error":   err.Error(),
			})
		}
		messages = append(messages, msg)
	}

	// we fetched newest-first (for skip/limit to work on the latest page),
	// reverse back to chronological order for display
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	if messages == nil {
		messages = []models.Message{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"msgs": messages,
	})
}

// GetUserUnreadedMsg
// @Summary Get unread message count & records for user
// @Description Get unread message count & records for user
// @Tags Chat
// @Accept json
// @Produce json
// @Param userid query string true "user id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /chat/get-user-unreadmsg [get]
func GetUserUnreadMsg(c *fiber.Ctx) error {

	var UnReadedMsgSchema = database.DB.Collection("unReadedmessages")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	queryUserid := c.Query("userid")
	if queryUserid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "user id query param is required",
		})
	}

	// a user can only check their own unread messages
	if queryUserid != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you are not allowed to view another user's messages",
		})
	}

	filter := bson.M{"mainUserid": queryUserid, "isReaded": false}

	cursor, err := UnReadedMsgSchema.Find(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to retrieve unread messages",
			"error":   err.Error(),
		})
	}
	defer cursor.Close(ctx)

	var urms []models.UnReadMsg
	totalUnreadMessageCount := 0

	for cursor.Next(ctx) {
		var urm models.UnReadMsg
		if err := cursor.Decode(&urm); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "failed to decode unread message",
				"error":   err.Error(),
			})
		}
		// filter already guarantees isReaded == false, so every doc here counts
		urms = append(urms, urm)
		totalUnreadMessageCount += urm.NumOfUnreadMessages
	}

	if urms == nil {
		urms = []models.UnReadMsg{}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"messages": urms,
		"total":    totalUnreadMessageCount,
	})
}

// MarkMsgAsReaded
// @Summary mark messages as read for user
// @Description mark messages as read for user, updates the record to isReaded=true, count=0
// @Tags Chat
// @Accept json
// @Produce json
// @Param mainuid query string true "main user id"
// @Param otheruid query string true "other user id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /chat/mark-msg-asreaded [get]
func MarkMsgAsReaded(c *fiber.Ctx) error {

	var UnReadedMsgSchema = database.DB.Collection("unReadedmessages")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	mainuid := c.Query("mainuid")
	otheruid := c.Query("otheruid")
	if mainuid == "" || otheruid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "mainuid and otheruid query params are required",
		})
	}

	// a user can only mark their own conversations as read
	if mainuid != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you are not allowed to modify another user's read status",
		})
	}

	filter := bson.M{"mainUserid": mainuid, "otherUserid": otheruid}
	update := bson.M{"$set": bson.M{"isReaded": true, "numOfUnreadedMessages": 0}}
	findOpts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var updatedDoc bson.M
	err := UnReadedMsgSchema.FindOneAndUpdate(ctx, filter, update, findOpts).Decode(&updatedDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "failed to mark message as read",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isMarked": true,
	})
}
