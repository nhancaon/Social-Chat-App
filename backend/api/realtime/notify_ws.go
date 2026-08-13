package realtime

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// HandleClient upgrades an authenticated request to a WebSocket the server
// pushes notifications through; the client isn't expected to send anything
// back other than protocol-level pings.
func (nm *NotificationManager) HandleClient(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return fiber.ErrUnauthorized
	}

	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	return websocket.New(func(conn *websocket.Conn) {
		conn.SetReadLimit(8192)
		nm.AddNotificationConnection(userID, conn)
		defer nm.RemoveNotificationConnection(userID)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})(c)
}
