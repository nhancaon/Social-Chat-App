package tests

import (
	"Server/database"
	"Server/routes"
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var app *fiber.App

func TestMain(m *testing.M) {
	// Setup
	setup()

	// Run Tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func setup() {
	// Load test environment variables
	if err := godotenv.Load("../.env.test"); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Printf("Warning: Could not load .env file: %v", err)
		} else {
			log.Println("Warning: Using .env instead of .env.test — make sure this is not production config")
		}
	}

	// Set JWT secret if not provided
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-jwt-secret-key")
	}

	// Connect to test db
	connectTestDB()

	// Setup fiber app
	app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))

	// Setup routes
	routes.SetupAuthRoutes(app)
	// ..
}

func connectTestDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use test db
	mongoUri := os.Getenv("TEST_MONGO_URI")
	if mongoUri == "" {
		mongoUri = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Fatal("Failed to connect to test db: ", err)
	}

	// Verify the connection actually works
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping test db: ", err)
	}

	database.Client = client
	database.DB = client.Database("social_test")
}

func cleanup() {
	if database.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Drop test db
		if err := database.DB.Drop(ctx); err != nil {
			log.Printf("Warning: failed to drop test db: %v", err)
		}
		if err := database.Client.Disconnect(ctx); err != nil {
			log.Printf("Warning: failed to disconnect test db: %v", err)
		}
	}
}

// cleanupCollections drops individual collections between tests
func cleanupCollections() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collections := []string{"users"}
	for _, collection := range collections {
		if err := database.DB.Collection(collection).Drop(ctx); err != nil {
			log.Printf("Warning: failed to drop collection %s: %v", collection, err)
		}
	}
}
