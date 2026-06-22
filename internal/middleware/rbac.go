package middleware

import (
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func RequirePermission(userRepo *repositories.UserRepository, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok || userRole != "admin" {
			return utils.Error(c, 403, "FORBIDDEN", "Admin access required")
		}

		userID, ok := c.Locals("userID").(string)
		if !ok {
			return utils.Error(c, 401, "UNAUTHORIZED", "User not found in context")
		}

		uid, err := uuid.Parse(userID)
		if err != nil {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID")
		}

		user, err := userRepo.FindByID(uid)
		if err != nil {
			return utils.Error(c, 401, "UNAUTHORIZED", "User not found")
		}

		if user.Role.Name == "super_admin" {
			return c.Next()
		}

		for _, perm := range user.Role.Permissions {
			if perm.Name == permission {
				return c.Next()
			}
		}

		return utils.Error(c, 403, "FORBIDDEN", "Permission denied: "+permission)
	}
}
