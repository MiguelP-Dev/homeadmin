package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear all env vars to ensure defaults are used
	keys := []string{"PORT", "ENV", "DATABASE_URL", "JWT_SECRET", "JWT_EXPIRATION_HOURS", "CORS_ORIGINS", "CSRF_KEY"}
	for _, k := range keys {
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected Port '8080', got '%s'", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env 'development', got '%s'", cfg.Env)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("expected empty DatabaseURL, got '%s'", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("expected empty JWTSecret, got '%s'", cfg.JWTSecret)
	}
	if cfg.JWTExpirationHours != 24 {
		t.Errorf("expected JWTExpirationHours 24, got %d", cfg.JWTExpirationHours)
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "http://localhost:8080" {
		t.Errorf("expected CORSOrigins ['http://localhost:8080'], got %v", cfg.CORSOrigins)
	}
	if cfg.CSRFKey != "" {
		t.Errorf("expected empty CSRFKey, got '%s'", cfg.CSRFKey)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("ENV", "production")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/mydb?sslmode=require")
	os.Setenv("JWT_SECRET", "super-secret-key")
	os.Setenv("JWT_EXPIRATION_HOURS", "48")
	os.Setenv("CSRF_KEY", "csrf-32-byte-key-here-12345678")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_EXPIRATION_HOURS")
		os.Unsetenv("CSRF_KEY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("expected Port '3000', got '%s'", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env 'production', got '%s'", cfg.Env)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/mydb?sslmode=require" {
		t.Errorf("expected correct DatabaseURL, got '%s'", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "super-secret-key" {
		t.Errorf("expected JWTSecret 'super-secret-key', got '%s'", cfg.JWTSecret)
	}
	if cfg.JWTExpirationHours != 48 {
		t.Errorf("expected JWTExpirationHours 48, got %d", cfg.JWTExpirationHours)
	}
	if cfg.CSRFKey != "csrf-32-byte-key-here-12345678" {
		t.Errorf("expected CSRFKey 'csrf-32-byte-key-here-12345678', got '%s'", cfg.CSRFKey)
	}
}

func TestCORSOriginsParsing(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single origin",
			value:    "http://localhost:8080",
			expected: []string{"http://localhost:8080"},
		},
		{
			name:     "multiple origins",
			value:    "http://localhost:8080,https://home.example.com,https://admin.example.com",
			expected: []string{"http://localhost:8080", "https://home.example.com", "https://admin.example.com"},
		},
		{
			name:     "origin with spaces after comma",
			value:    "http://localhost:8080, https://home.example.com",
			expected: []string{"http://localhost:8080", " https://home.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CORS_ORIGINS", tt.value)
			defer os.Unsetenv("CORS_ORIGINS")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(cfg.CORSOrigins) != len(tt.expected) {
				t.Fatalf("expected %d origins, got %d", len(tt.expected), len(cfg.CORSOrigins))
			}

			for i, origin := range cfg.CORSOrigins {
				if origin != tt.expected[i] {
					t.Errorf("origin[%d]: expected '%s', got '%s'", i, tt.expected[i], origin)
				}
			}
		})
	}
}
