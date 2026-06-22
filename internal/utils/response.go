package utils

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Message    string      `json:"message,omitempty"`
	Error      *ErrorData  `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ErrorData struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func Success(c *fiber.Ctx, data interface{}, message string) error {
	return c.JSON(Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func SuccessWithPagination(c *fiber.Ctx, data interface{}, page, limit int, total int64) error {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return c.JSON(Response{
		Success: true,
		Data:    data,
		Pagination: &Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func Error(c *fiber.Ctx, status int, code string, message interface{}) error {
	return c.Status(status).JSON(Response{
		Success: false,
		Error: &ErrorData{
			Code:    code,
			Message: message,
		},
	})
}

func IsDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}
