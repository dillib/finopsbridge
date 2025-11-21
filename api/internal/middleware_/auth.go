package middleware

import (
	"encoding/json"
	"finopsbridge/api/internal/config_"
	"strings"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gofiber/fiber/v2"
)

func ClerkAuth(secretKey string) fiber.Handler {
	client, err := clerk.NewClient(secretKey)
	if err != nil {
		panic(err)
	}

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		sessionClaims, err := client.VerifyToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Store user info in context
		userID, _ := sessionClaims.Get("sub")
		orgID, _ := sessionClaims.Get("org_id")

		c.Locals("userID", userID)
		c.Locals("orgID", orgID)
		c.Locals("sessionClaims", sessionClaims)

		return c.Next()
	}
}

func GetUserID(c *fiber.Ctx) string {
	if userID, ok := c.Locals("userID").(string); ok {
		return userID
	}
	return ""
}

func GetOrgID(c *fiber.Ctx) string {
	if orgID, ok := c.Locals("orgID").(string); ok {
		return orgID
	}
	return ""
}

