package handlers

import (
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type PermissionHandler struct {
	repo *repositories.PermissionRepository
}

func NewPermissionHandler(repo *repositories.PermissionRepository) *PermissionHandler {
	return &PermissionHandler{repo: repo}
}

type CreatePermissionInput struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (h *PermissionHandler) GetAll(c *fiber.Ctx) error {
	permissions, err := h.repo.FindAll()
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch permissions")
	}
	return utils.Success(c, permissions, "")
}

func (h *PermissionHandler) Create(c *fiber.Ctx) error {
	var input CreatePermissionInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	perm := &models.Permission{
		Name:        input.Name,
		Description: input.Description,
	}

	if err := h.repo.Create(perm); err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", "Permission already exists")
		}
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to create permission")
	}

	return utils.Success(c, perm, "Permission created")
}
