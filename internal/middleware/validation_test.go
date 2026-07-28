package middleware

import (
	"errors"
	"strings"
	"testing"
)

// 4.4 — ValidateRequired with empty value should return error with field name.
func TestValidateRequired_EmptyValue(t *testing.T) {
	vErr := ValidateRequired("", "name")
	if vErr == nil {
		t.Fatal("ValidateRequired(\"\", \"name\") = nil, want non-nil ValidationError")
	}
	if vErr.Field != "name" {
		t.Errorf("Field = %q, want %q", vErr.Field, "name")
	}
	if vErr.Message == "" {
		t.Error("Message should not be empty")
	}
}

// 4.5 — ValidateRequired with valid value should return nil.
func TestValidateRequired_ValidValue(t *testing.T) {
	vErr := ValidateRequired("ok", "name")
	if vErr != nil {
		t.Errorf("ValidateRequired(\"ok\", \"name\") = %v, want nil", vErr)
	}
}

// 4.6 — ValidatePositive with negative value should return error.
func TestValidatePositive_NegativeValue(t *testing.T) {
	vErr := ValidatePositive(-5, "amount")
	if vErr == nil {
		t.Fatal("ValidatePositive(-5, \"amount\") = nil, want non-nil ValidationError")
	}
	if vErr.Field != "amount" {
		t.Errorf("Field = %q, want %q", vErr.Field, "amount")
	}
}

// 4.7 — ValidateIn with value not in set should return error.
func TestValidateIn_InvalidValue(t *testing.T) {
	vErr := ValidateIn("invalid", "category", []string{"food", "rent"})
	if vErr == nil {
		t.Fatal("ValidateIn(\"invalid\", ...) = nil, want non-nil ValidationError")
	}
	if vErr.Field != "category" {
		t.Errorf("Field = %q, want %q", vErr.Field, "category")
	}
}

// 4.8 — Validate collector with multiple errors should return combined AppError.
func TestValidate_MultipleErrors(t *testing.T) {
	err := Validate(
		ValidateRequired("", "desc"),
		ValidatePositive(-1, "amt"),
	)
	if err == nil {
		t.Fatal("Validate() = nil, want non-nil error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *AppError", err)
	}
	if appErr.Status != 400 {
		t.Errorf("Status = %d, want 400", appErr.Status)
	}

	msg := appErr.Error()
	if !strings.Contains(msg, "desc") {
		t.Errorf("error message should contain field \"desc\": %s", msg)
	}
	if !strings.Contains(msg, "amt") {
		t.Errorf("error message should contain field \"amt\": %s", msg)
	}
}

// --- Triangulation tests ---

func TestValidateRequired_Whitespace(t *testing.T) {
	vErr := ValidateRequired("   ", "name")
	if vErr == nil {
		t.Error("ValidateRequired(\"   \", \"name\") = nil, want non-nil (whitespace should fail)")
	}
}

func TestValidateRequired_NilValue(t *testing.T) {
	vErr := ValidateRequired(nil, "name")
	if vErr == nil {
		t.Error("ValidateRequired(nil, \"name\") = nil, want non-nil")
	}
}

func TestValidatePositive_Zero(t *testing.T) {
	vErr := ValidatePositive(0, "amount")
	if vErr == nil {
		t.Error("ValidatePositive(0, \"amount\") = nil, want non-nil (zero is not a valid amount)")
	}
}

func TestValidatePositive_Positive(t *testing.T) {
	vErr := ValidatePositive(100, "amount")
	if vErr != nil {
		t.Errorf("ValidatePositive(100, \"amount\") = %v, want nil", vErr)
	}
}

func TestValidateIn_ValidValue(t *testing.T) {
	vErr := ValidateIn("food", "category", []string{"food", "rent"})
	if vErr != nil {
		t.Errorf("ValidateIn(\"food\", ...) = %v, want nil", vErr)
	}
}

func TestValidateMinLength_TooShort(t *testing.T) {
	vErr := ValidateMinLength("ab", "password", 8)
	if vErr == nil {
		t.Fatal("ValidateMinLength(\"ab\", \"password\", 8) = nil, want non-nil")
	}
	if vErr.Field != "password" {
		t.Errorf("Field = %q, want %q", vErr.Field, "password")
	}
}

func TestValidateMinLength_ExactLength(t *testing.T) {
	vErr := ValidateMinLength("12345678", "password", 8)
	if vErr != nil {
		t.Errorf("ValidateMinLength(\"12345678\", \"password\", 8) = %v, want nil", vErr)
	}
}

func TestValidateMaxLength_TooLong(t *testing.T) {
	long := strings.Repeat("a", 256)
	vErr := ValidateMaxLength(long, "description", 255)
	if vErr == nil {
		t.Fatal("ValidateMaxLength(256-char, \"description\", 255) = nil, want non-nil")
	}
	if vErr.Field != "description" {
		t.Errorf("Field = %q, want %q", vErr.Field, "description")
	}
}

func TestValidateMaxLength_ExactLength(t *testing.T) {
	exact := strings.Repeat("a", 255)
	vErr := ValidateMaxLength(exact, "description", 255)
	if vErr != nil {
		t.Errorf("ValidateMaxLength(255-char, \"description\", 255) = %v, want nil", vErr)
	}
}

func TestValidate_AllNil(t *testing.T) {
	err := Validate(
		ValidateRequired("ok", "name"),
		ValidatePositive(10, "amount"),
	)
	if err != nil {
		t.Errorf("Validate(all valid) = %v, want nil", err)
	}
}

func TestValidate_SingleError(t *testing.T) {
	err := Validate(
		ValidateRequired("", "email"),
	)
	if err == nil {
		t.Fatal("Validate(single error) = nil, want non-nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *AppError", err)
	}
	if !strings.Contains(appErr.Error(), "email") {
		t.Errorf("error message should contain \"email\": %s", appErr.Error())
	}
}
