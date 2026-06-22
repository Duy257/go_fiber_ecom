package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service *services.OrderService
}

func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

type CancelOrderInput struct {
	Note string `json:"note"`
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return utils.Error(c, 401, "UNAUTHORIZED", "User not authenticated")
	}

	var input services.CreateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	input.CustomerID = userID

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	order, err := h.service.Create(input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order created")
}

func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	order, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Order not found")
	}

	return utils.Success(c, order, "")
}

func (h *OrderHandler) GetMyOrders(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return utils.Error(c, 401, "UNAUTHORIZED", "User not authenticated")
	}

	customerID, err := uuid.Parse(userID)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID in token")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	orders, total, err := h.service.GetByCustomerID(customerID, page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch orders")
	}

	return utils.SuccessWithPagination(c, orders, page, limit, total)
}

func (h *OrderHandler) GetByShop(c *fiber.Ctx) error {
	shopIDStr := c.Query("shop_id")
	if shopIDStr == "" {
		return utils.Error(c, 400, "VALIDATION_ERROR", "shop_id query param is required")
	}

	shopID, err := uuid.Parse(shopIDStr)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop ID")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	status := c.Query("status")

	var st *string
	if status != "" {
		st = &status
	}

	orders, total, err := h.service.GetByShopID(shopID, st, page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch orders")
	}

	return utils.SuccessWithPagination(c, orders, page, limit, total)
}

func (h *OrderHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateOrderStatusInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	order, err := h.service.UpdateStatus(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order status updated")
}

func (h *OrderHandler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input CancelOrderInput
	if err := c.BodyParser(&input); err != nil {
		input.Note = "Cancelled"
	}
	if input.Note == "" {
		input.Note = "Cancelled"
	}

	order, err := h.service.Cancel(id, input.Note)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, order, "Order cancelled")
}
