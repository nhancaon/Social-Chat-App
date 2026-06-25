package controllers

import (
	"Server/database"
	"Server/models"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Register
// @Summary Register a new user
// @Description Register an new user by providing email, password, first name, last name
// @Tags Authentication
// @Accept json
// @Produce json
// @Param user body models.CreateUser true "user register details"
// @Success 201 {object} models.UserModel
// @Failure 400 {object} map[string]interface{}
// @Router /user/signup [post]
func Register(c *fiber.Ctx) error {
	UserSchema := database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var body models.CreateUser
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Check user đã tồn tại chưa
	var existingUser models.UserModel
	err := UserSchema.FindOne(ctx, bson.M{"email": body.Email}).Decode(&existingUser)
	if err == nil {
		// tìm thấy document -> email đã tồn tại
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User with email " + body.Email + " already exists",
		})
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		fmt.Println("FindOne error (register):", err)
		// lỗi thật (DB down, timeout...) -> không nên cho qua
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing user",
		})
	}

	// hashing password
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	newUser := models.UserModel{
		Name:      body.FirstName + " " + body.LastName,
		Email:     body.Email,
		Password:  string(hashPassword),
		Followers: make([]string, 0),
		Following: make([]string, 0),
	}

	result, err := UserSchema.InsertOne(ctx, &newUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Không cần query lại DB — gán trực tiếp ID vừa được Mongo sinh ra
	newUser.ID = result.InsertedID.(primitive.ObjectID)
	newUser.Password = ""

	// create the token
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": newUser.ID.Hex(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	token, err := claims.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"result": newUser,
		"token":  token,
	})
}

// Login
// @Summary Login a user
// @Description Login a user by providing email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param user body models.LoginUser true "user login details"
// @Success 201 {object} models.UserModel
// @Failure 400 {object} map[string]interface{}
// @Router /user/signin [post]
func Login(c *fiber.Ctx) error {
	UserSchema := database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var body models.LoginUser
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Check user đã tồn tại chưa
	var existingUser models.UserModel
	err := UserSchema.FindOne(ctx, bson.M{"email": body.Email}).Decode(&existingUser)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// không tìm thấy document -> email chưa tồn tại
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Invalid user with email " + body.Email,
		})
	}
	if err != nil {
		fmt.Println("FindOne error (login):", err)
		// lỗi thật (DB down, timeout...) -> không nên cho qua
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing user",
		})
	}

	// check password
	checkPass := bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(body.Password))
	if checkPass != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid password",
		})
	}

	existingUser.Password = "" // remove password from response

	// create the token
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": existingUser.ID.Hex(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	token, err := claims.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"result": existingUser,
		"token":  token,
	})
}
