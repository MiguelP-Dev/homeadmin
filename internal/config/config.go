package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// minProductionJWTSecretLen is the minimum accepted JWT_SECRET length when
// ENV=production. Shorter secrets are brute-forceable and must not boot.
const minProductionJWTSecretLen = 16

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	DBDriver           string
	JWTSecret          string
	JWTExpirationHours int
	CORSOrigins        []string
	CSRFKey            string
}

func Load() (*Config, error) {
	godotenv.Load() // Ignore error if .env doesn't exist (cloud envs)

	expiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		DBDriver:           getEnv("DB_DRIVER", "sqlite"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTExpirationHours: expiration,
		CORSOrigins:        strings.Split(getEnv("CORS_ORIGINS", "http://localhost:8080"), ","),
		CSRFKey:            os.Getenv("CSRF_KEY"),
	}

	if cfg.Env == "production" && len(cfg.JWTSecret) < minProductionJWTSecretLen {
		return nil, fmt.Errorf(
			"config: JWT_SECRET must be set to at least %d characters when ENV=production (got %d characters)",
			minProductionJWTSecretLen, len(cfg.JWTSecret),
		)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
