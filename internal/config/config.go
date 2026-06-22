package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	ServerPort    string
	AdminEmail    string
	AdminPassword string
	AdminPhone    string
}

func Load() *Config {
	godotenv.Load()

	accessTTL, _ := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	refreshTTL, _ := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))

	cfg := &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "go_fiber"),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		JWTAccessTTL:  accessTTL,
		JWTRefreshTTL: refreshTTL,
		ServerPort:    getEnv("SERVER_PORT", "3000"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		AdminPhone:    getEnv("ADMIN_PHONE", "0900000000"),
	}

	if cfg.JWTSecret == "secret" || len(cfg.JWTSecret) < 16 {
		log.Fatal("JWT_SECRET must be at least 16 characters and not the default value")
	}
	if cfg.AdminPassword == "admin123" {
		log.Fatal("ADMIN_PASSWORD must be changed from the default value")
	}
	if cfg.JWTAccessTTL == 0 || cfg.JWTRefreshTTL == 0 {
		log.Fatal("JWT_ACCESS_TTL and JWT_REFRESH_TTL must be valid durations")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
