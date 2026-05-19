package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

var validate = validator.New()

// CustomValidator — daftarkan ke Echo agar bisa pakai c.Validate()
type CustomValidator struct{}

func NewValidator() *CustomValidator {
	return &CustomValidator{}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := validate.Struct(i); err != nil {
		// Format error menjadi pesan yang ramah pengguna
		var messages []string
		for _, e := range err.(validator.ValidationErrors) {
			messages = append(messages, formatValidationError(e))
		}
		return echo.NewHTTPError(http.StatusBadRequest, strings.Join(messages, "; "))
	}
	return nil
}

func formatValidationError(e validator.FieldError) string {
	field := strings.ToLower(e.Field())
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("field '%s' wajib diisi", field)
	case "email":
		return fmt.Sprintf("field '%s' harus berupa email yang valid", field)
	case "min":
		return fmt.Sprintf("field '%s' minimal %s karakter", field, e.Param())
	case "max":
		return fmt.Sprintf("field '%s' maksimal %s karakter", field, e.Param())
	case "url":
		return fmt.Sprintf("field '%s' harus berupa URL yang valid", field)
	default:
		return fmt.Sprintf("field '%s' tidak valid", field)
	}
}
