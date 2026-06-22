package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RoleHandler struct {
	service *services.RoleService
}

func NewRoleHandler(service *services.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) GetAll(c *fiber.Ctx) error {
	roles, err := h.service.GetAll()
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch roles")
	}
	return utils.Success(c, roles, "")
}

func (h *RoleHandler) Create(c *fiber.Ctx) error {
	var input services.CreateRoleInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	role, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", "Role already exists")
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, role, "Role created")
}

func (h *RoleHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateRoleInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	role, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, role, "Role updated")
}

func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Role deleted")
}
