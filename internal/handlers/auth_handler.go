package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginInput struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) RegisterCustomer(c *fiber.Ctx) error {
	var input services.CreateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	customer, tokens, err := h.authService.RegisterCustomer(input)
	if err != nil {
		if utils.IsDuplicateEntry(err) {
			return utils.Error(c, 409, "DUPLICATE_ENTRY", "Email or phone already exists")
		}
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, fiber.Map{
		"customer": customer,
		"tokens":   tokens,
	}, "Registration successful")
}

func (h *AuthHandler) LoginAdmin(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	tokens, err := h.authService.LoginAdmin(input.Login, input.Password)
	if err != nil {
		return utils.Error(c, 401, "INVALID_CREDENTIALS", err.Error())
	}

	return utils.Success(c, tokens, "Login successful")
}

func (h *AuthHandler) LoginCustomer(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	tokens, err := h.authService.LoginCustomer(input.Login, input.Password)
	if err != nil {
		return utils.Error(c, 401, "INVALID_CREDENTIALS", err.Error())
	}

	return utils.Success(c, tokens, "Login successful")
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var input RefreshInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", errs)
	}

	accessToken, err := h.authService.Refresh(input.RefreshToken)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", err.Error())
	}

	return utils.Success(c, fiber.Map{"access_token": accessToken}, "Token refreshed")
}
