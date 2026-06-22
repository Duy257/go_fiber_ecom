package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShippingHandler struct {
	service *services.ShippingService
}

func NewShippingHandler(service *services.ShippingService) *ShippingHandler {
	return &ShippingHandler{service: service}
}

func (h *ShippingHandler) Estimate(c *fiber.Ctx) error {
	var input services.ShippingEstimateInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	shopID, err := uuid.Parse(input.ShopID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid shop_id")
	}

	result, err := h.service.Calculate(shopID, input.ShippingLatitude, input.ShippingLongitude)
	if err != nil {
		code := "VALIDATION_ERROR"
		status := 400
		switch err.Error() {
		case "SHOP_LOCATION_NOT_SET":
			code = "SHOP_LOCATION_NOT_SET"
		case "OUTSIDE_DELIVERY_RANGE":
			code = "OUTSIDE_DELIVERY_RANGE"
		case "SHIPPING_CONFIG_NOT_FOUND":
			code = "SHIPPING_CONFIG_NOT_FOUND"
			status = 500
		}
		return utils.Error(c, status, code, err.Error())
	}

	return utils.Success(c, result, "")
}

func (h *ShippingHandler) GetConfig(c *fiber.Ctx) error {
	result, err := h.service.GetConfig()
	if err != nil {
		return utils.Error(c, 500, "SHIPPING_CONFIG_NOT_FOUND", "Shipping config not found")
	}
	return utils.Success(c, result, "")
}

func (h *ShippingHandler) UpdateConfig(c *fiber.Ctx) error {
	var input services.UpdateShippingConfigInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	result, err := h.service.UpdateConfig(input)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", err.Error())
	}

	return utils.Success(c, result, "Shipping config updated")
}
