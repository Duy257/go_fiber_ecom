package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field())
		switch e.Tag() {
		case "required":
			errors[field] = fmt.Sprintf("%s is required", field)
		case "email":
			errors[field] = fmt.Sprintf("%s must be a valid email", field)
		case "min":
			errors[field] = fmt.Sprintf("%s must be at least %s characters", field, e.Param())
		default:
			errors[field] = fmt.Sprintf("%s is invalid", field)
		}
	}
	return errors
}

func IsValidEmailOrPhone(login string) bool {
	if strings.Contains(login, "@") {
		return validate.Var(login, "email") == nil
	}
	return len(login) >= 9 && len(login) <= 15
}
