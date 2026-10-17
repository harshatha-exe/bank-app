package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DatabaseURL        string
	Port               string
	CSRFSecret         string
	SessionIdleTimeout time.Duration
	SessionMaxLifetime time.Duration
	PasswordMinLength  int
}

func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	csrfSecret := os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		return Config{}, fmt.Errorf("CSRF_SECRET is required")
	}

	return Config{
		DatabaseURL:        databaseURL,
		Port:               port,
		CSRFSecret:         csrfSecret,
		SessionIdleTimeout: envDuration("SESSION_IDLE_TIMEOUT", 30*time.Minute),
		SessionMaxLifetime: envDuration("SESSION_MAX_LIFETIME", 24*time.Hour),
		PasswordMinLength:  envInt("PASSWORD_MIN_LENGTH", 12),
	}, nil
}

func envDuration(name string, fallback time.Duration) time.Duration {
	if value := os.Getenv(name); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func envInt(name string, fallback int) int {
	if value := os.Getenv(name); value != "" {
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}
