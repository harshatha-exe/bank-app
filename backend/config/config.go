package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL        string
	Port               string
	CSRFSecret         string
	SessionIdleTimeout time.Duration
	SessionMaxLifetime time.Duration
	PasswordMinLength  int
	MaxTransferPaise   int64
	TrustedProxies     []string
	TransferRateLimit  int
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
	trustedProxies, err := envTrustedProxies()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:        databaseURL,
		Port:               port,
		CSRFSecret:         csrfSecret,
		SessionIdleTimeout: envDuration("SESSION_IDLE_TIMEOUT", 30*time.Minute),
		SessionMaxLifetime: envDuration("SESSION_MAX_LIFETIME", 24*time.Hour),
		PasswordMinLength:  envInt("PASSWORD_MIN_LENGTH", 12),
		MaxTransferPaise:   envMoneyPaise("MAX_TRANSFER_AMOUNT_INR", 10000000),
		TrustedProxies:     trustedProxies,
		TransferRateLimit:  envInt("TRANSFER_RATE_LIMIT", 10),
	}, nil
}

func envTrustedProxies() ([]string, error) {
	value := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if value == "" {
		return nil, nil
	}
	var proxies []string
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("TRUSTED_PROXIES contains an empty entry")
		}
		if entry == "0.0.0.0/0" || entry == "::/0" {
			return nil, fmt.Errorf("TRUSTED_PROXIES must not trust all addresses")
		}
		if ip := net.ParseIP(entry); ip == nil {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid TRUSTED_PROXIES entry %q", entry)
			}
			ones, _ := network.Mask.Size()
			if ones == 0 && network.IP.IsUnspecified() {
				return nil, fmt.Errorf("TRUSTED_PROXIES must not trust all addresses")
			}
		}
		proxies = append(proxies, entry)
	}
	return proxies, nil
}

func envMoneyPaise(name string, fallback int64) int64 {
	if value := os.Getenv(name); value != "" {
		var whole int64
		if _, err := fmt.Sscanf(value, "%d", &whole); err == nil && whole > 0 {
			return whole * 100
		}
	}
	return fallback
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
