package realtime

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

func NewApp(ws map[string]*websocket.Conn, wsMu *sync.Mutex) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))

	app.Get("/ws/:userId", websocket.New(func(c *websocket.Conn) {
		userID := c.Params("userId")
		log.Printf("User %s connected", userID)

		wsMu.Lock()
		ws[userID] = c
		wsMu.Unlock()

		defer func() {
			log.Printf("User %s disconnected", userID)
			wsMu.Lock()
			if existing, ok := ws[userID]; ok && existing == c {
				delete(ws, userID)
			}
			wsMu.Unlock()
			c.Close()
		}()

		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(25 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					wsMu.Lock()
					err := c.WriteMessage(websocket.PingMessage, nil)
					wsMu.Unlock()
					if err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}()
		defer close(done)

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				log.Printf("User %s connection closed: %v", userID, err)
				break
			}
		}
	}))

	return app
}
