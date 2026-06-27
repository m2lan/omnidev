// Package validator provides request validation utilities.
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// Validate is the global validator instance.
var Validate *validator.Validate

func init() {
	Validate = validator.New()

	// Register custom validators
	_ = Validate.RegisterValidation("password", validatePassword)
	_ = Validate.RegisterValidation("slug", validateSlug)
	_ = Validate.RegisterValidation("uuid_or_empty", validateUUIDOrEmpty)
}

// ValidateStruct validates a struct and returns user-friendly errors.
func ValidateStruct(s interface{}) map[string]string {
	err := Validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		field := strings.ToLower(err.Field())
		switch err.Tag() {
		case "required":
			errors[field] = fmt.Sprintf("%s is required", err.Field())
		case "email":
			errors[field] = fmt.Sprintf("%s must be a valid email address", err.Field())
		case "min":
			errors[field] = fmt.Sprintf("%s must be at least %s characters", err.Field(), err.Param())
		case "max":
			errors[field] = fmt.Sprintf("%s must be at most %s characters", err.Field(), err.Param())
		case "oneof":
			errors[field] = fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
		case "password":
			errors[field] = fmt.Sprintf("%s must contain at least 8 characters with uppercase, lowercase, digit, and special character", err.Field())
		case "slug":
			errors[field] = fmt.Sprintf("%s must contain only lowercase letters, numbers, and hyphens", err.Field())
		default:
			errors[field] = fmt.Sprintf("%s failed validation: %s", err.Field(), err.Tag())
		}
	}

	return errors
}

// validatePassword checks if a password meets complexity requirements.
// At least 8 chars, 1 uppercase, 1 lowercase, 1 digit, 1 special char.
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// validateSlug checks if a string is a valid slug (lowercase, numbers, hyphens).
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if slug == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
	return matched
}

// validateUUIDOrEmpty accepts either a valid UUID or an empty string.
func validateUUIDOrEmpty(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, val)
	return matched
}
