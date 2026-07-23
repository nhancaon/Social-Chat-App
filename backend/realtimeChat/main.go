package main

import (
	"log"
	"time"

	"realTimeChat/realtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// allowedOrigins should list your actual frontend domain(s).
// Wildcard + AllowCredentials is unsafe for production.
var allowedOrigins = map[string]bool{
	"https://your-frontend.example.com": true,
}

func main() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOriginsFunc: func(origin string) bool {
			return allowedOrigins[origin]
		},
	}))

	manager := realtime.NewConnectionManager(realtime.GetUserFriends)

	// register ws route
	app.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		id := c.Params("id")

		// TODO: verify the caller actually owns this id (e.g. validate a
		// JWT/session token passed as a query param or header) before
		// trusting it. As written, anyone can connect as any user id.

		if manager == nil {
			log.Printf("ConnectionManager is nil")
		}

		manager.AddConnection(id, c)
		defer func() {
			manager.RemoveConnection(id)
			c.Close()
		}()

		// panic recovery so a bad message can't take down the whole process
		defer func() {
			if r := recover(); r != nil {
				log.Printf("recovered from panic in ws handler for %s: %v", id, r)
			}
		}()

		// keepalive: bail out if the client goes silent
		const pongWait = 60 * time.Second
		c.SetReadDeadline(time.Now().Add(pongWait))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		for {
			var msg realtime.Message // declared inside the loop: avoids stale fields
			err := c.ReadJSON(&msg)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("connection closed normally for %s", id)
				} else {
					handleWebSocketError(err, id)
				}
				manager.RemoveConnection(id)
				c.Close()
				break
			}

			if msg.Sender == "" || msg.Receiver == "" || msg.Content == "" {
				log.Printf("dropping invalid message from %s: missing fields", id)
				continue
			}

			log.Printf("Received message from %s to %s: %s", msg.Sender, msg.Receiver, msg.Content)
			manager.SendToReceiver(msg)
		}
	}))

	log.Fatal(app.Listen(":8001"))
}

func handleWebSocketError(err error, userID string) {
	log.Printf("WebSocket error for user %s: %v", userID, err)
}
