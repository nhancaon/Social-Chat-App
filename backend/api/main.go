package main

import (
	"Server/database"
	"Server/kafka"
	"Server/realtime"
	"Server/server"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
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
	//load env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables instead")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	nodeID := flag.String("node_id", "", "Unique identifier for this node")
	kafkaAddr := flag.String("kafka", "127.0.0.1:29092", "kafka address")
	flag.Parse()

	resolvedNodeID := *nodeID
	if resolvedNodeID == "" {
		if hostname, err := os.Hostname(); err == nil {
			resolvedNodeID = hostname
		} else {
			resolvedNodeID = fmt.Sprintf("node-%d", time.Now().UnixNano())
		}
	}
	log.Printf("node_id=%s kafka=%s", resolvedNodeID, *kafkaAddr)

	database.Connect()
	database.InitRedis()

	if err := kafka.WaitForKafka(*kafkaAddr, 1*time.Minute); err != nil {
		log.Fatalf("kafka not ready: %v", err)
	}
	time.Sleep(3 * time.Second) // give topic metadata time to propagate across brokers

	chatHub, err := realtime.InitChatHub(*kafkaAddr, resolvedNodeID)
	if err != nil {
		log.Fatalf("failed to start chat hub: %v", err)
	}

	if err := realtime.InitNotificationHub(*kafkaAddr, resolvedNodeID); err != nil {
		log.Fatalf("failed to start notification manager: %v", err)
	}

	heartbeatMgr, err := kafka.NewHeartbeatManager(*kafkaAddr, resolvedNodeID, chatHub)
	if err != nil {
		log.Fatalf("failed to start heartbeat manager: %v", err)
	}

	app := server.NewHTTPServer()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"node":   resolvedNodeID,
		})
	})

	go func() {
		log.Printf("HTTP server running on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

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

	heartbeatMgr.Stop()
	chatHub.Close()
	if notificationHub := realtime.GetNotificationHub(); notificationHub != nil {
		notificationHub.Close()
	}
	database.CloseRedis()

	log.Println("Servers exited cleanly")
}
