package controllers

import (
	"Server/database"
	"Server/models"
	"context"
	"math"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Create Post
// @Summary create a new post
// @Description create new post
// @Tags Posts
// @Accept json
// @Produce json
// @Param post body models.CreateOrUpdatePost true "post create details"
// @Success 201 {object} models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts [post]
func CreatePost(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	var PostSchema = database.DB.Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var body models.CreateOrUpdatePost
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
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
			"error": err.Error(),
		})
	}

	// start set data
	var post models.PostModel
	post.Creator = userID
	post.Likes = make([]string, 0)
	post.Comments = make([]string, 0)
	post.CreatedAt = time.Now()
	post.Title = body.Title
	post.Message = body.Message
	post.SelectedFile = body.SelectedFile
	post.Name = user.Name
	// set data end

	result, err := PostSchema.InsertOne(ctx, &post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var createdPost models.PostModel
	query := bson.M{"_id": result.InsertedID}
	if err := PostSchema.FindOne(ctx, query).Decode(&createdPost); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdPost)
}

// Get Post
// @Summary Get a post
// @Description Get a post by id
// @Tags Posts
// @Accept json
// @Produce json
// @Param id path string true "Post id"
// @Success 200 {object} models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Router /posts/{id} [get]
func GetPost(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "post id is required",
		})
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	var post models.PostModel
	query := bson.M{"_id": objID}

	if err := PostSchema.FindOne(ctx, query).Decode(&post); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "post not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"post": post,
	})
}

// Update Post
// @Summary Update post
// @Description Update post
// @Tags Posts
// @Accept json
// @Produce json
// @Param id path string true "Post Id"
// @Param post body models.CreateOrUpdatePost true "update post details"
// @Success 200 {object} models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/{id} [patch]
func UpdatePost(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	primID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	var newData models.CreateOrUpdatePost
	if err := c.BodyParser(&newData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// authorization start
	var authPost models.PostModel
	if err := PostSchema.FindOne(ctx, bson.M{"_id": primID}).Decode(&authPost); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "post not found",
		})
	}

	if authPost.Creator != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to update this post.",
		})
	}
	// authorization end

	// set data
	authPost.Title = newData.Title
	authPost.Message = newData.Message
	authPost.SelectedFile = newData.SelectedFile

	_, err = PostSchema.UpdateOne(ctx, bson.M{"_id": authPost.ID}, bson.M{"$set": bson.M{
		"title":        authPost.Title,
		"message":      authPost.Message,
		"selectedFile": authPost.SelectedFile,
	}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": authPost})
}

// GetAllPosts Post
// @Summary Get All Posts
// @Description GetAllPosts with pagination
// @Tags Posts
// @Accept json
// @Produce json
// @Param page query int false "page number"
// @Param id query string true "user id"
// @Success 200 {object} []models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts [get]
func GetAllPosts(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	var userSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.UserModel
	var posts []models.PostModel

	userid := c.Query("id")
	if userid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user id is required",
		})
	}

	MainUserid, err := primitive.ObjectIDFromHex(userid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	if err := userSchema.FindOne(ctx, bson.M{"_id": MainUserid}).Decode(&user); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	// get user following list ids and add our user id to it
	following := append(user.Following, userid)

	const LIMIT = 2

	findOptions := options.Find()
	filter := bson.M{"creator": bson.M{"$in": following}}

	total, err := PostSchema.CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	findOptions.SetSkip((int64(page) - 1) * int64(LIMIT))
	findOptions.SetLimit(int64(LIMIT))
	findOptions.SetSort(bson.D{{Key: "_id", Value: -1}})

	cursor, err := PostSchema.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var post models.PostModel
		if err := cursor.Decode(&post); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	if posts == nil {
		posts = make([]models.PostModel, 0)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":          posts,
		"currentPage":   page,
		"numberOfPages": math.Ceil(float64(total) / float64(LIMIT)),
	})
}

// GetPostsUsersBySearch Post
// @Summary Get Posts and users by search query
// @Description get posts and users matching the search query
// @Tags Posts
// @Accept json
// @Produce json
// @Param searchQuery query string true "Search query"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/search [get]
func GetPostsUsersBySearch(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	var userSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var users []models.UserModel
	var posts []models.PostModel

	filterPost := bson.M{}
	filterUser := bson.M{}

	search := c.Query("searchQuery")
	if search != "" {
		// escape regex special chars to avoid regex injection / ReDoS
		safe := regexp.QuoteMeta(search)

		filterPost = bson.M{
			"$or": []bson.M{
				{"title": bson.M{"$regex": primitive.Regex{Pattern: safe, Options: "i"}}},
				{"message": bson.M{"$regex": primitive.Regex{Pattern: safe, Options: "i"}}},
			},
		}

		filterUser = bson.M{
			"$or": []bson.M{
				{"name": bson.M{"$regex": primitive.Regex{Pattern: safe, Options: "i"}}},
				{"email": bson.M{"$regex": primitive.Regex{Pattern: safe, Options: "i"}}},
			},
		}
	}

	cursorPosts, err := PostSchema.Find(ctx, filterPost, options.Find())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer cursorPosts.Close(ctx)

	cursorUsers, err := userSchema.Find(ctx, filterUser, options.Find())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer cursorUsers.Close(ctx)

	for cursorUsers.Next(ctx) {
		var user models.UserModel
		if err := cursorUsers.Decode(&user); err != nil {
			continue
		}
		users = append(users, user)
	}

	for cursorPosts.Next(ctx) {
		var post models.PostModel
		if err := cursorPosts.Decode(&post); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	if users == nil {
		users = make([]models.UserModel, 0)
	}
	if posts == nil {
		posts = make([]models.PostModel, 0)
	}

	return c.JSON(fiber.Map{
		"user":  users,
		"posts": posts,
	})
}

// Comment Post
// @Summary comment post
// @Description comment post
// @Tags Posts
// @Accept json
// @Produce json
// @Param id path string true "Post Id"
// @Param post body models.ComnmentPost true "comment value"
// @Success 200 {object} models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/{id}/commentPost [post]
func CommentPost(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	var UserSchema = database.DB.Collection("users")
	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	postid, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	var b models.ComnmentPost
	if err := c.BodyParser(&b); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	var post models.PostModel
	if err := PostSchema.FindOne(ctx, bson.M{"_id": postid}).Decode(&post); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "post not found",
		})
	}

	// append comment atomically to avoid lost updates
	_, err = PostSchema.UpdateOne(ctx, bson.M{"_id": postid}, bson.M{"$push": bson.M{"comments": b.Value}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := PostSchema.FindOne(ctx, bson.M{"_id": postid}).Decode(&post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// create notification (best-effort, doesn't fail the comment if it errors)
	objId, err := primitive.ObjectIDFromHex(userID)
	if err == nil {
		var user models.UserModel
		if err := UserSchema.FindOne(ctx, bson.M{"_id": objId}).Decode(&user); err == nil {
			notification := models.Notification{
				MainUID:   post.Creator,
				TargetID:  postid.Hex(),
				Deatils:   user.Name + " Commented on your Post",
				User:      models.User{Name: user.Name, Avatart: user.ImageUrl},
				CreatedAt: time.Now(),
			}
			if _, err := NotificationSchema.InsertOne(ctx, notification); err != nil {
				// log and continue, the comment itself already succeeded
				// (use your logger here if available)
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": post,
	})
}

// like Post
// @Summary like or unlike a post
// @Description Like or unlike a post by its id
// @Tags Posts
// @Accept json
// @Produce json
// @Param id path string true "Post Id"
// @Success 200 {object} models.PostModel
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/{id}/likePost [patch]
func LikePost(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	var UserSchema = database.DB.Collection("users")
	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "you are not authorized",
		})
	}

	postid, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	var post models.PostModel
	if err := PostSchema.FindOne(ctx, bson.M{"_id": postid}).Decode(&post); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "post not found",
		})
	}

	alreadyLiked := slices.Contains(post.Likes, userID)

	if alreadyLiked {
		// atomic remove, avoids lost-update races between concurrent likes
		_, err = PostSchema.UpdateOne(ctx, bson.M{"_id": postid}, bson.M{"$pull": bson.M{"likes": userID}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	} else {
		// atomic add (also guards against duplicate likes from a race)
		_, err = PostSchema.UpdateOne(ctx, bson.M{"_id": postid}, bson.M{"$addToSet": bson.M{"likes": userID}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// create notification (best-effort)
		objId, err := primitive.ObjectIDFromHex(userID)
		if err == nil {
			var user models.UserModel
			if err := UserSchema.FindOne(ctx, bson.M{"_id": objId}).Decode(&user); err == nil {
				notification := models.Notification{
					MainUID:   post.Creator,
					TargetID:  post.ID.Hex(),
					Deatils:   user.Name + " Liked your Post",
					User:      models.User{Name: user.Name, Avatart: user.ImageUrl},
					CreatedAt: time.Now(),
				}
				if _, err := NotificationSchema.InsertOne(ctx, notification); err != nil {
					// log and continue, the like itself already succeeded
				}
			}
		}
	}

	if err := PostSchema.FindOne(ctx, bson.M{"_id": postid}).Decode(&post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"post": post,
	})
}

// Delete Post
// @Summary Delete post by id
// @Description Delete post by post id, needs auth token of post creator
// @Tags Posts
// @Accept json
// @Produce json
// @Param id path string true "Post Id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/{id} [delete]
func DeletePost(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	primID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	// authorization start
	var authPost models.PostModel
	if err := PostSchema.FindOne(ctx, bson.M{"_id": primID}).Decode(&authPost); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "post not found",
		})
	}

	if authPost.Creator != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to delete this post.",
		})
	}
	// authorization end

	result, err := PostSchema.DeleteOne(ctx, bson.M{"_id": primID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if result.DeletedCount == 1 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Post deleted successfully!",
		})
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"message": "can't delete post!",
	})
}
