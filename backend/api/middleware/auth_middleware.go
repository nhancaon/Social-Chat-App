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
	userID, err := ParseUserToken(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "unauthenticated",
		})
	}

	c.Locals("userId", userID)
	return c.Next()
}

// ParseUserToken validates tokenStr (no "Bearer " prefix) and returns the
// user id carried in its "sub" claim. Shared by the HTTP/WS middlewares
// above and by any handler that needs to re-validate a token directly
// (e.g. a token-refresh endpoint).
func ParseUserToken(tokenStr string) (string, error) {
	if tokenStr == "" {
		return "", errors.New("missing token")
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
		return "", errors.New("invalid or expired token")
	}

	claims, ok := parsedToken.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", errors.New("invalid token claims")
	}

	return claims.Subject, nil
}
