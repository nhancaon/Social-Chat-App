package main

import (
	"Server/database"
	_ "Server/docs"
	"Server/gapi"
	pb "Server/protos"
	"Server/routes"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// @title Fiber Golang Rest API
// @version 1.0
// @description this is  Swagger docs for Social chat API created with golang and fiber
// @host localhost:5000
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	//load env example
	// if err := godotenv.Load(".env.example"); err != nil {
	// 	log.Println("No .env.example file found, using system environment variables instead")
	// }

	//load env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables instead")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "5001"
	}

	database.Connect()
	app := fiber.New()

	app.Use(cors.New(
		cors.Config{
			// AllowOrigins:     "*",
			// AllowHeaders:     "Content-Type, Authorization",
			// AllowMethods:     "GET, POST, PUT, DELETE",
			AllowCredentials: true,
			AllowOriginsFunc: func(origin string) bool {
				return true
			},
		},
	))

	// Setup Grpc Server
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRealtimeChatServiceServer(grpcServer, &gapi.Server{})
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server running on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	// ---- end gRPC setup ----

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to Socail app")
	})

	//Setup routes
	routes.SetupAuthRoutes(app)
	routes.SetupUserRoutes(app)
	routes.SetupPostRoutes(app)
	routes.SetupChatRoutes(app)
	routes.SetupNotificationRoutes(app)

	//Server swagger docs
	app.Get("/swagger/*", swagger.HandlerDefault)

	// ---- Start HTTP server (non-blocking) ----
	go func() {
		log.Printf("HTTP server running on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()
	// ---- end HTTP server start ----

	// ---- Graceful shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()

	log.Println("Servers exited cleanly")
}
