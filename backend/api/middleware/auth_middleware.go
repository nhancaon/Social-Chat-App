package middleware

import (
	"errors"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	var tokenStr string
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		tokenStr = strings.TrimSpace(authHeader)
	}

	return authenticate(c, tokenStr)
}

// WSAuthMiddleware authenticates a WebSocket upgrade request. Browsers'
// native WebSocket client cannot set an Authorization header, so it also
// accepts the token as a "token" query parameter, falling back to the header
// for non-browser clients.
func WSAuthMiddleware(c *fiber.Ctx) error {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.Get("Authorization")
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authenticate(c, tokenStr)
}

func authenticate(c *fiber.Ctx, tokenStr string) error {
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "unauthenticated",
		})
	}

	secretKey := os.Getenv("JWT_SECRET")

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Enforce expected signing method to prevent algorithm confusion attacks
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil || !parsedToken.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "unauthenticated",
		})
	}

	claims, ok := parsedToken.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "unauthenticated",
		})
	}

	c.Locals("userId", claims.Subject)
	return c.Next()
}
