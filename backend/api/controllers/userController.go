package controllers

import (
	"Server/database"
	"Server/models"
	"context"
	"slices"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetUserBy ID
// @Summary Get User By ID
// @Description GetUser Deatils By ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 201 {object} models.UserModel
// @Failure 400 {object} map[string]interface{}
// @Router /user/getUser/{id} [get]
func GetUserByID(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	var PostSchema = database.DB.Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.UserModel
	var posts []models.PostModel

	objId, _ := primitive.ObjectIDFromHex(c.Params("id"))
	strID := c.Params("id")
	// GET and REturn user posts
	findOptions := options.Find()
	postResult, err := PostSchema.Find(ctx, bson.M{"creator": strID}, findOptions)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err,
		})
	}

	defer postResult.Close(ctx)
	for postResult.Next(ctx) {
		var singlePost models.PostModel
		postResult.Decode(&singlePost)
		posts = append(posts, singlePost)
	}

	if posts == nil {
		posts = make([]models.PostModel, 0)
	}
	// get user data
	userResult := UserSchema.FindOne(ctx, bson.M{"_id": objId})

	if userResult.Err() != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"success": false,
			"message": "User Not found",
		})
	}

	userResult.Decode(&user)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":  user,
		"posts": posts,
	})
}

// UpdateUser
// @Summary update user data
// @Description update user deatils
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body models.UpdateUser true "deatils "
// @Success 201 {object} models.UserModel
// @Failure 400 {object} map[string]interface{}
// @security BearerAuth
// @Router /user/Update/{id} [patch]
func UpdateUser(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//
	extUid := c.Locals("userId").(string)

	if extUid != c.Params("id") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "You Are Not Authroized to Update This Profile",
		})
	}

	userid, _ := primitive.ObjectIDFromHex(c.Params("id"))

	var user models.UpdateUser
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":   "Invalid request body",
			"deatils": err.Error(),
		})
	}

	update := bson.M{"name": user.Name, "imageUrl": user.ImageUrl, "bio": user.Bio}

	result, err := UserSchema.UpdateOne(ctx, bson.M{"_id": userid}, bson.M{"$set": update})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "cannot update the user data",
			"deatils": err.Error(),
		})
	}
	//
	var updateUsser models.UserModel
	if result.MatchedCount == 1 {
		err := UserSchema.FindOne(ctx, bson.M{"_id": userid}).Decode(&updateUsser)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"deatils": err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": updateUsser})

}

// Following Users
// @Summary Follow/UnFollow User
// @Description follow or un follow a user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @security BearerAuth
// @Router /user/{id}/following [patch]
func FollowingUser(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	var NotificationSchema = database.DB.Collection("notifications")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fuid := c.Params("id")
	suid := c.Locals("userId").(string)

	if fuid == suid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "cannot follow yourself",
		})
	}

	FirstUserID, err := primitive.ObjectIDFromHex(fuid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "invalid user id",
		})
	}
	SecondUserID, err := primitive.ObjectIDFromHex(suid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "invalid current user id",
		})
	}

	var FirstUser models.UserModel
	var SecondUser models.UserModel

	if err := UserSchema.FindOne(ctx, bson.M{"_id": FirstUserID}).Decode(&FirstUser); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"details": err.Error(),
		})
	}
	if err := UserSchema.FindOne(ctx, bson.M{"_id": SecondUserID}).Decode(&SecondUser); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"details": err.Error(),
		})
	}

	isFollowing := slices.Contains(FirstUser.Followers, suid)

	if isFollowing {
		// UNFOLLOW — xóa khỏi cả 2 chiều, atomic bằng $pull
		_, err = UserSchema.UpdateOne(ctx, bson.M{"_id": FirstUserID},
			bson.M{"$pull": bson.M{"followers": suid}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
		_, err = UserSchema.UpdateOne(ctx, bson.M{"_id": SecondUserID},
			bson.M{"$pull": bson.M{"following": fuid}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
	} else {
		// FOLLOW — thêm vào cả 2 chiều, atomic bằng $addToSet (tự tránh duplicate)
		_, err = UserSchema.UpdateOne(ctx, bson.M{"_id": FirstUserID},
			bson.M{"$addToSet": bson.M{"followers": suid}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
		_, err = UserSchema.UpdateOne(ctx, bson.M{"_id": SecondUserID},
			bson.M{"$addToSet": bson.M{"following": fuid}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}

		notification := models.Notification{
			MainUID:   FirstUser.ID.Hex(),
			TargetID:  SecondUser.ID.Hex(),
			Deatils:   SecondUser.Name + " Start Following You!",
			User:      models.User{Name: SecondUser.Name, Avatart: SecondUser.ImageUrl},
			CreatedAt: time.Now(),
		}
		if _, err := NotificationSchema.InsertOne(ctx, notification); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to create notification",
				"error":   err.Error(),
			})
		}
	}

	// Lấy lại data mới nhất sau khi update
	if err := UserSchema.FindOne(ctx, bson.M{"_id": FirstUserID}).Decode(&FirstUser); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"details": err.Error(),
		})
	}
	if err := UserSchema.FindOne(ctx, bson.M{"_id": SecondUserID}).Decode(&SecondUser); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"SecondUser": SecondUser,
		"FirstUser":  FirstUser,
	})
}

// // GetSugUser Users
// // @Summary Get Suggersted users
// // @Description get suggested userses based on the current user's following list
// // @Tags Users
// // @Accept json
// // @Produce json
// // @Param id query string true "User ID"
// // @Success 200 {object} map[string]interface{}
// // @Failure 400 {object} map[string]interface{}
// // @security BearerAuth
// // @Router /user/getSug [get]
func GetSugUser(c *fiber.Ctx) error {

	var UserSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var MainUser models.UserModel
	var AllSugUsers []models.UserModel

	mainUserHex := c.Query("id")
	MainUserID, err := primitive.ObjectIDFromHex(mainUserHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"details": err.Error(),
		})
	}

	if err := UserSchema.FindOne(ctx, bson.M{"_id": MainUserID}).Decode(&MainUser); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"details": err.Error(),
		})
	}

	// Convert MainUser.Following sang ObjectID (1 lần, để dùng cho query $in)
	var followingObjIDs []primitive.ObjectID
	for _, fid := range MainUser.Following {
		if oid, err := primitive.ObjectIDFromHex(fid); err == nil {
			followingObjIDs = append(followingObjIDs, oid)
		}
	}

	var FollowingUsers []models.UserModel
	if len(followingObjIDs) > 0 {
		cursor, err := UserSchema.Find(ctx, bson.M{"_id": bson.M{"$in": followingObjIDs}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &FollowingUsers); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
	}

	// Set để tra cứu nhanh + tự loại trùng (thay cho slices.Contains trong loop)
	alreadyFollowing := make(map[string]bool)
	for _, fid := range MainUser.Following {
		alreadyFollowing[fid] = true
	}

	sugSet := make(map[string]bool) // dùng map để tránh trùng, đỡ phải gọi Contains() từng lần (O(1) thay vì O(n))

	addCandidate := func(id string) {
		if id == mainUserHex { // không gợi ý chính mình
			return
		}
		if alreadyFollowing[id] { // đã follow rồi, không cần gợi ý
			return
		}
		sugSet[id] = true
	}

	for _, u := range FollowingUsers {
		for _, id := range u.Following {
			addCandidate(id)
		}
		for _, id := range u.Followers {
			addCandidate(id)
		}
	}

	// Convert sugSet -> []ObjectID để query 1 lần
	var sugObjIDs []primitive.ObjectID
	for id := range sugSet {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			sugObjIDs = append(sugObjIDs, oid)
		}
	}

	AllSugUsers = make([]models.UserModel, 0)
	if len(sugObjIDs) > 0 {
		cursor, err := UserSchema.Find(ctx, bson.M{"_id": bson.M{"$in": sugObjIDs}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &AllSugUsers); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"details": err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"users": AllSugUsers})
}

// DeleteUser
// @Summary delete user
// @Description delete user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object}  map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @security BearerAuth
// @Router /user/delete/{id} [delete]
func DeleteUser(c *fiber.Ctx) error {
	var UserSchema = database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//
	extUid := c.Locals("userId").(string)

	if extUid != c.Params("id") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "You Are Not Authroized to Delete This User",
		})
	}

	userID, err := primitive.ObjectIDFromHex(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid User id",
		})
	}

	result, err := UserSchema.DeleteOne(ctx, bson.M{"_id": userID})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "faild to delete user",
			"error":   err.Error(),
		})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}
	// sucuss
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "User Deleted Successfully!",
	})
}
