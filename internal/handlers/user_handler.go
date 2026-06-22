package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	users, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch users")
	}

	return utils.SuccessWithPagination(c, users, page, limit, total)
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	user, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "User not found")
	}

	return utils.Success(c, user, "")
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var input services.CreateUserInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	user, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", "Email or phone already exists")
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, user, "User created")
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	user, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, user, "User updated")
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "User deleted")
}
