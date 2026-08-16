package middleware

import (
	"fmt"
	"strings"
)

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field   string
	Message string
	Key     string
	Args    []any
}

// ValidateRequired checks that a string or interface{} value is non-empty and non-whitespace.
// Returns *ValidationError if invalid, nil if valid.
func ValidateRequired(value interface{}, fieldName string) *ValidationError {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s is required", fieldName),
				Key:     "validation.required",
				Args:    []any{fieldName},
			}
		}
	case nil:
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s is required", fieldName),
			Key:     "validation.required",
			Args:    []any{fieldName},
		}
	}
	return nil
}

// ValidateMinLength checks that a string value has at least min characters.
func ValidateMinLength(value, fieldName string, min int) *ValidationError {
	if len([]rune(value)) < min {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s must be at least %d characters", fieldName, min),
			Key:     "validation.min_length",
			Args:    []any{fieldName},
		}
	}
	return nil
}

// ValidateMaxLength checks that a string value has at most max characters.
func ValidateMaxLength(value, fieldName string, max int) *ValidationError {
	if len([]rune(value)) > max {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s must be at most %d characters", fieldName, max),
			Key:     "validation.max_length",
			Args:    []any{fieldName},
		}
	}
	return nil
}

// ValidatePositive checks that a numeric value is greater than zero.
// Accepts int, int64, float64.
func ValidatePositive(value interface{}, fieldName string) *ValidationError {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s must be positive", fieldName),
				Key:     "validation.positive",
				Args:    []any{fieldName},
			}
		}
	case int64:
		if v <= 0 {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s must be positive", fieldName),
				Key:     "validation.positive",
				Args:    []any{fieldName},
			}
		}
	case float64:
		if v <= 0 {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s must be positive", fieldName),
				Key:     "validation.positive",
				Args:    []any{fieldName},
			}
		}
	}
	return nil
}

// ValidateEmailFormat checks that a string looks like a basic email (contains @ and .).
func ValidateEmailFormat(value, fieldName string) *ValidationError {
	if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s must be a valid email address", fieldName),
			Key:     "validation.email",
			Args:    []any{fieldName},
		}
	}
	return nil
}

// ValidateIn checks that a string value is one of the allowed set.
func ValidateIn(value, fieldName string, set []string) *ValidationError {
	for _, allowed := range set {
		if value == allowed {
			return nil
		}
	}
	return &ValidationError{
		Field:   fieldName,
		Message: fmt.Sprintf("%s must be one of: %s", fieldName, strings.Join(set, ", ")),
		Key:     "validation.in",
		Args:    []any{fieldName},
	}
}

// Validate collects validation results and returns a combined error if any failed.
// Returns nil if all validators returned nil, or an AppError with status 400
// containing the expense.validation_failed key for i18n and raw messages for backward compat.
func Validate(rules ...*ValidationError) error {
	var messages []string
	for _, v := range rules {
		if v != nil {
			messages = append(messages, v.Message)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return &AppError{
		Status:  400,
		Key:     "expense.validation_failed",
		Message: strings.Join(messages, "; "),
	}
}
