package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"realTimeNotification/gapi"
	"realTimeNotification/realtime"

	"github.com/gofiber/websocket/v2"
)

func main() {
	var wsMu sync.Mutex
	ws := make(map[string]*websocket.Conn)

	// Start the gRPC server (receives notification requests from other
	// backend services, e.g. the main REST API).
	grpcServer, err := gapi.StartGRPCServer(ws, &wsMu)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}

	// Start the WebSocket server (browsers connect here to receive
	// notifications in real time).
	wsApp := realtime.NewApp(ws, &wsMu)
	go func() {
		log.Println("WebSocket server running on port 8088")
		if err := wsApp.Listen(":8088"); err != nil {
			log.Printf("WebSocket server stopped: %v", err)
		}
	}()

	// Block until we receive a shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := wsApp.ShutdownWithContext(ctx); err != nil {
		log.Printf("WebSocket server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()

	log.Println("Servers exited cleanly")
}
