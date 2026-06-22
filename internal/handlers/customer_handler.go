package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CustomerHandler struct {
	service *services.CustomerService
}

func NewCustomerHandler(service *services.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	customers, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch customers")
	}

	return utils.SuccessWithPagination(c, customers, page, limit, total)
}

func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	customer, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Customer not found")
	}

	return utils.Success(c, customer, "")
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var input services.CreateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	customer, err := h.service.Create(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", "Email or phone already exists")
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Customer created")
}

func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	customer, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Customer updated")
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Customer deleted")
}

func (h *CustomerHandler) GetProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid user ID")
	}

	customer, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Customer not found")
	}

	return utils.Success(c, customer, "")
}

func (h *CustomerHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid user ID")
	}

	var input services.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	customer, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Profile updated")
}
