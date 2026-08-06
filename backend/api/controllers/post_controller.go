package controllers

import (
	"Server/database"
	"Server/gapi"
	"Server/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// commentsLookupStage joins each post with its comments (from the separate
// "comments" collection) and, for every comment, the commenter's name/avatar.
// Shared by every handler that returns a post to the frontend so the
// "comments" field always has the same enriched shape.
func commentsLookupStage() bson.M {
	return bson.M{
		"$lookup": bson.M{
			"from": "comments",
			"let":  bson.M{"postId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$postId", "$$postId"}}}},
				{"$sort": bson.M{"createdAt": -1}},
				{"$lookup": bson.M{
					"from": "users",
					"let":  bson.M{"uid": "$userId"},
					"pipeline": []bson.M{
						{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$uid"}}}},
						{"$project": bson.M{"name": 1, "imageUrl": 1}},
					},
					"as": "user",
				}},
				{"$unwind": bson.M{"path": "$user", "preserveNullAndEmptyArrays": true}},
			},
			"as": "comments",
		},
	}
}

// fetchPostWithComments returns a single post populated with its comments.
func fetchPostWithComments(ctx context.Context, postSchema *mongo.Collection, filter bson.M) (models.PostModel, error) {
	pipeline := []bson.M{
		{"$match": filter},
		commentsLookupStage(),
		{"$limit": 1},
	}

	cursor, err := postSchema.Aggregate(ctx, pipeline)
	if err != nil {
		return models.PostModel{}, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return models.PostModel{}, mongo.ErrNoDocuments
	}

	var post models.PostModel
	if err := cursor.Decode(&post); err != nil {
		return models.PostModel{}, err
	}
	return post, nil
}

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

	createdPost, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": result.InsertedID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// new post changes both the creator's own profile post list and their
	// own feed (their feed includes their own posts); other followers'
	// feed caches are left to expire via TTL - fanning out to every
	// follower's cache on each post isn't worth it at this scale
	database.InvalidateCacheByPattern(ctx, fmt.Sprintf("user:profile:%s:page:*", userID))
	database.InvalidateCacheByPattern(ctx, fmt.Sprintf("posts:feed:%s:page:*", userID))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": createdPost,
	})

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

	post, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": objID})
	if err != nil {
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

	updatedPost, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": authPost.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	database.InvalidateCacheByPattern(ctx, fmt.Sprintf("user:profile:%s:page:*", userID))
	database.InvalidateCacheByPattern(ctx, fmt.Sprintf("posts:feed:%s:page:*", userID))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": updatedPost})
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

	// feed content depends on who the user follows, so it can only be
	// invalidated cheaply for the user themselves (on follow/unfollow or
	// their own post create/update/delete). A new post from someone they
	// follow can't feasibly invalidate every follower's cache, so this is
	// a short-TTL microcache: it absorbs bursts of repeat requests but
	// still goes stale on its own within a few seconds.
	cacheKey := fmt.Sprintf("posts:feed:%s:page:%d", userid, page)
	if cachedData, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
		var cachedRes models.CachedGetAllPostResponse
		if err := json.Unmarshal([]byte(cachedData), &cachedRes); err == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data":          cachedRes.Data,
				"currentPage":   cachedRes.CurrentPage,
				"numberOfPages": cachedRes.NumberOfPages,
				"cached":        true,
			})
		}
	}

	if err := userSchema.FindOne(ctx, bson.M{"_id": MainUserid}).Decode(&user); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	// get user following list ids and add our user id to it
	following := append(user.Following, userid)

	const LIMIT = 2

	filter := bson.M{"creator": bson.M{"$in": following}}

	total, err := PostSchema.CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	pipeline := []bson.M{
		{"$match": filter},
		{"$sort": bson.M{"_id": -1}},
		{"$skip": (int64(page) - 1) * int64(LIMIT)},
		{"$limit": int64(LIMIT)},
		commentsLookupStage(),
	}

	cursor, err := PostSchema.Aggregate(ctx, pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &posts); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if posts == nil {
		posts = make([]models.PostModel, 0)
	}

	numberOfPages := math.Ceil(float64(total) / float64(LIMIT))

	response := models.CachedGetAllPostResponse{
		Data:          posts,
		CurrentPage:   page,
		NumberOfPages: numberOfPages,
	}
	if responseJSON, err := json.Marshal(response); err != nil {
		log.Printf("failed to marshal feed response for caching: %v", err)
	} else if err := database.RedisClient.Set(ctx, cacheKey, responseJSON, 15*time.Second).Err(); err != nil {
		log.Printf("failed to set feed cache for user %s: %v", userid, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":          posts,
		"currentPage":   page,
		"numberOfPages": numberOfPages,
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
				{"name": bson.M{"$regex": primitive.Regex{Pattern: safe, Options: "i"}}},
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
		"data": fiber.Map{
			"users": users,
			"posts": posts,
		},
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
	var CommentSchema = database.DB.Collection("comments")
	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
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

	// comments live in their own collection (not embedded in the post
	// document) so a post with many comments never risks hitting MongoDB's
	// 16MB document limit, and concurrent comments can never overwrite
	// each other the way a whole-document $push/$set race could.
	comment := models.Comment{
		ID:        primitive.NewObjectID(),
		PostID:    postid,
		UserID:    userObjID,
		Value:     b.Value,
		CreatedAt: time.Now(),
	}
	if _, err := CommentSchema.InsertOne(ctx, comment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// create notification (best-effort, doesn't fail the comment if it errors)
	if post.Creator != userID {
		var user models.UserModel
		if err := UserSchema.FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err == nil {
			notification := models.Notification{
				MainUID:   post.Creator,
				TargetID:  postid.Hex(),
				UserID:    user.ID.Hex(),
				Details:   user.Name + " Commented on your Post",
				User:      models.User{Name: user.Name, Avatar: user.ImageUrl},
				CreatedAt: time.Now(),
			}
			res, err := NotificationSchema.InsertOne(ctx, notification)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Failed to create notification",
					"error":   err.Error(),
				})
			}
			notification.ID = res.InsertedID.(primitive.ObjectID)
			//call grpc function to send notification to the user
			gapi.SendNotification(notification)
		}
	}

	updatedPost, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": postid})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updatedPost,
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
		if post.Creator != userID {
			objId, err := primitive.ObjectIDFromHex(userID)
			if err == nil {
				var user models.UserModel
				if err := UserSchema.FindOne(ctx, bson.M{"_id": objId}).Decode(&user); err == nil {
					notification := models.Notification{
						MainUID:   post.Creator,
						TargetID:  post.ID.Hex(),
						UserID:    user.ID.Hex(),
						Details:   user.Name + " Liked your Post",
						User:      models.User{Name: user.Name, Avatar: user.ImageUrl},
						CreatedAt: time.Now(),
					}

					res, err := NotificationSchema.InsertOne(ctx, notification)
					if err != nil {
						return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
							"message": "Failed to create notification",
							"error":   err.Error(),
						})
					}
					notification.ID = res.InsertedID.(primitive.ObjectID)
					//call grpc function to send notification to the user
					gapi.SendNotification(notification)
				}
			}
		}
	}

	updatedPost, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": postid})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updatedPost,
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
	var CommentSchema = database.DB.Collection("comments")
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
		// comments now live in their own collection, so deleting the post
		// no longer cascades automatically - clean up its comments here to
		// avoid leaving orphaned documents behind
		if _, err := CommentSchema.DeleteMany(ctx, bson.M{"postId": primID}); err != nil {
			log.Printf("failed to delete comments for post %s: %v", primID.Hex(), err)
		}

		database.InvalidateCacheByPattern(ctx, fmt.Sprintf("user:profile:%s:page:*", userID))
		database.InvalidateCacheByPattern(ctx, fmt.Sprintf("posts:feed:%s:page:*", userID))

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Post deleted successfully!",
		})
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"message": "can't delete post!",
	})
}

// Delete Comment
// @Summary Delete a comment
// @Description Delete a comment by id. Only the comment's author or the post's creator may delete it
// @Tags Posts
// @Accept json
// @Produce json
// @Param postId path string true "Post Id"
// @Param commentId path string true "Comment Id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security BearerAuth
// @Router /posts/{postId}/comments/{commentId} [delete]
func DeleteComment(c *fiber.Ctx) error {

	var PostSchema = database.DB.Collection("posts")
	var CommentSchema = database.DB.Collection("comments")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "You are not authorized",
		})
	}

	postID, err := primitive.ObjectIDFromHex(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post id",
		})
	}

	commentID, err := primitive.ObjectIDFromHex(c.Params("commentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid comment id",
		})
	}

	var comment models.Comment
	if err := CommentSchema.FindOne(ctx, bson.M{"_id": commentID, "postId": postID}).Decode(&comment); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "comment not found",
		})
	}

	var post models.PostModel
	if err := PostSchema.FindOne(ctx, bson.M{"_id": postID}).Decode(&post); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "post not found",
		})
	}

	// only the comment's author or the post's creator can remove it
	if comment.UserID.Hex() != userID && post.Creator != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not authorized to delete this comment.",
		})
	}

	result, err := CommentSchema.DeleteOne(ctx, bson.M{"_id": commentID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "comment not found or already deleted",
		})
	}

	updatedPost, err := fetchPostWithComments(ctx, PostSchema, bson.M{"_id": postID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updatedPost,
	})
}
