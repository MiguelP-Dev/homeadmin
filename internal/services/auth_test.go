package services

import (
	"strings"
	"testing"
)

// --- HashPassword tests (spec §1.26) ---

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("expected hash to start with $2a$, got prefix: %s", hash[:7])
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	hash1, _ := HashPassword("secret123")
	hash2, _ := HashPassword("secret123")
	if hash1 == hash2 {
		t.Error("two calls to HashPassword with same input produced identical hashes (bcrypt salt randomization broken)")
	}
}

// --- CheckPassword tests (spec §1.27-1.28) ---

func TestCheckPassword_Correct(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if !CheckPassword("secret123", hash) {
		t.Error("CheckPassword should return true for correct password")
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if CheckPassword("wrongpass", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestCheckPassword_Empty(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

// --- CreateToken tests (spec §1.29) ---

func TestCreateToken_Success(t *testing.T) {
	var householdID uint = 42
	token, err := CreateToken(1, &householdID, "member", "user@example.com", false, "test-secret", 24)
	if err != nil {
		t.Fatalf("CreateToken returned unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token string")
	}
}

func TestCreateToken_NilHousehold(t *testing.T) {
	token, err := CreateToken(1, nil, "admin", "admin@example.com", true, "test-secret", 24)
	if err != nil {
		t.Fatalf("CreateToken returned unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token string")
	}
}

// --- ValidateToken tests (spec §1.30) ---

func TestValidateToken_Valid(t *testing.T) {
	var householdID uint = 42
	token, err := CreateToken(1, &householdID, "member", "user@example.com", false, "test-secret", 24)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	claims, err := ValidateToken(token, "test-secret")
	if err != nil {
		t.Fatalf("ValidateToken returned unexpected error: %v", err)
	}
	if claims == nil {
		t.Fatal("ValidateToken returned nil claims")
	}
	if claims.UserID != 1 {
		t.Errorf("expected UserID=1, got %d", claims.UserID)
	}
	if claims.HouseholdID == nil || *claims.HouseholdID != 42 {
		t.Errorf("expected HouseholdID=42, got %v", claims.HouseholdID)
	}
	if claims.Role != "member" {
		t.Errorf("expected Role=member, got %s", claims.Role)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected Email=user@example.com, got %s", claims.Email)
	}
	if claims.IsAdmin {
		t.Errorf("expected IsAdmin=false, got %v", claims.IsAdmin)
	}
}

func TestValidateToken_NilHousehold(t *testing.T) {
	token, err := CreateToken(1, nil, "admin", "admin@example.com", true, "test-secret", 24)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	claims, err := ValidateToken(token, "test-secret")
	if err != nil {
		t.Fatalf("ValidateToken returned unexpected error: %v", err)
	}
	if claims.HouseholdID != nil {
		t.Errorf("expected nil HouseholdID, got %v", *claims.HouseholdID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", claims.Role)
	}
	if claims.Email != "admin@example.com" {
		t.Errorf("expected Email=admin@example.com, got %s", claims.Email)
	}
	if !claims.IsAdmin {
		t.Errorf("expected IsAdmin=true, got %v", claims.IsAdmin)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := CreateToken(1, nil, "member", "user@example.com", false, "correct-secret", 24)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	claims, err := ValidateToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("ValidateToken should return error for wrong secret")
	}
	if claims != nil {
		t.Error("ValidateToken should return nil claims on error")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	token, err := CreateToken(1, nil, "member", "user@example.com", false, "test-secret", 0)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	claims, err := ValidateToken(token, "test-secret")
	if err == nil {
		t.Fatal("ValidateToken should return error for expired token")
	}
	if claims != nil {
		t.Error("ValidateToken should return nil claims on error")
	}
}

func TestValidateToken_EmptyString(t *testing.T) {
	claims, err := ValidateToken("", "test-secret")
	if err == nil {
		t.Fatal("ValidateToken should return error for empty token string")
	}
	if claims != nil {
		t.Error("ValidateToken should return nil claims on error")
	}
}
