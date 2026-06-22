package middleware

import (
	"strings"

	"go-fiber/internal/config"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid authorization format")
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid or expired token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid token claims")
		}

		tokenType, _ := claims["type"].(string)
		if tokenType != "access" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid token type")
		}

		sub, _ := claims["sub"].(string)
		role, _ := claims["role"].(string)

		c.Locals("userID", sub)
		c.Locals("userRole", role)

		return c.Next()
	}
}
