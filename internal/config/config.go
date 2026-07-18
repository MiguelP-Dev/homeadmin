package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	CORSOrigins        []string
	CSRFKey            string
}

func Load() (*Config, error) {
	godotenv.Load() // Ignore error if .env doesn't exist (cloud envs)

	expiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))

	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTExpirationHours: expiration,
		CORSOrigins:        strings.Split(getEnv("CORS_ORIGINS", "http://localhost:8080"), ","),
		CSRFKey:            os.Getenv("CSRF_KEY"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
