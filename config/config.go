// Package config loads all application configuration from environment variables
// following 12-factor app principles. No defaults are assumed for secrets.
package config

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	App        AppConfig
	DB         DBConfig
	JWT        JWTConfig
	CORS       CORSConfig
	RateLimit  RateLimitConfig
	Migrations MigrationsConfig
}

// AppConfig holds HTTP server settings.
type AppConfig struct {
	Env  string
	Port string
	Name string
}

// DBConfig holds MySQL connection settings.
type DBConfig struct {
	Host               string
	Port               string
	Name               string
	User               string
	Password           string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
} 

// JWTConfig holds JWT signing keys and token parameters.
type JWTConfig struct {
	PrivateKey          *rsa.PrivateKey
	PublicKey           *rsa.PublicKey
	Issuer              string
	Audience            string
	AccessTokenTTL      time.Duration
	RefreshTokenTTLDays int
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins []string
}

// RateLimitConfig holds rate limiter settings for sensitive endpoints.
type RateLimitConfig struct {
	RPS   float64
	Burst int
}

// MigrationsConfig holds migration source path.
type MigrationsConfig struct {
	Path string
}

// Load reads all configuration from environment variables and returns a Config.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{}

	// App
	cfg.App.Env = getEnv("APP_ENV", "development")
	cfg.App.Port = getEnv("APP_PORT", "8080")
	cfg.App.Name = getEnv("APP_NAME", "business-tracker-backend")

	// DB
	cfg.DB.Host = requireEnv("DB_HOST")
	cfg.DB.Port = getEnv("DB_PORT", "3306")
	cfg.DB.Name = requireEnv("DB_NAME")
	cfg.DB.User = requireEnv("DB_USER")
	cfg.DB.Password = requireEnv("DB_PASSWORD")
	cfg.DB.MaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	cfg.DB.MaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 10)
	cfg.DB.ConnMaxLifetimeMin = getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)

	// JWT keys
	privateKey, publicKey, err := loadRSAKeys()
	if err != nil {
		return nil, fmt.Errorf("loading RSA keys: %w", err)
	}
	cfg.JWT.PrivateKey = privateKey
	cfg.JWT.PublicKey = publicKey
	cfg.JWT.Issuer = requireEnv("JWT_ISSUER")
	cfg.JWT.Audience = requireEnv("JWT_AUDIENCE")
	cfg.JWT.AccessTokenTTL = time.Duration(getEnvInt("JWT_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	cfg.JWT.RefreshTokenTTLDays = getEnvInt("JWT_REFRESH_TOKEN_TTL_DAYS", 7)

	// CORS
	originsRaw := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.CORS.AllowedOrigins = strings.Split(originsRaw, ",")

	// Rate limit
	cfg.RateLimit.RPS = getEnvFloat("RATE_LIMIT_RPS", 5)
	cfg.RateLimit.Burst = getEnvInt("RATE_LIMIT_BURST", 10)

	// Migrations
	cfg.Migrations.Path = getEnv("MIGRATIONS_PATH", "file://migrations")

	return cfg, nil
}

// DSN returns the MySQL data source name for database/sql.
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&multiStatements=true",
		c.User, c.Password, c.Host, c.Port, c.Name,
	)
}

// loadRSAKeys loads RSA keys from base64-encoded env vars (JWT_PRIVATE_KEY_B64,
// JWT_PUBLIC_KEY_B64) or falls back to file paths (JWT_PRIVATE_KEY_FILE,
// JWT_PUBLIC_KEY_FILE).
func loadRSAKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	var privPEM, pubPEM []byte

	if b64 := os.Getenv("JWT_PRIVATE_KEY_B64"); b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, nil, fmt.Errorf("decoding JWT_PRIVATE_KEY_B64: %w", err)
		}
		privPEM = decoded
	} else if path := os.Getenv("JWT_PRIVATE_KEY_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("reading JWT_PRIVATE_KEY_FILE: %w", err)
		}
		privPEM = data
	} else {
		return nil, nil, errors.New("JWT_PRIVATE_KEY_B64 or JWT_PRIVATE_KEY_FILE must be set")
	}

	if b64 := os.Getenv("JWT_PUBLIC_KEY_B64"); b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, nil, fmt.Errorf("decoding JWT_PUBLIC_KEY_B64: %w", err)
		}
		pubPEM = decoded
	} else if path := os.Getenv("JWT_PUBLIC_KEY_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("reading JWT_PUBLIC_KEY_FILE: %w", err)
		}
		pubPEM = data
	} else {
		return nil, nil, errors.New("JWT_PUBLIC_KEY_B64 or JWT_PUBLIC_KEY_FILE must be set")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing RSA private key: %w", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing RSA public key: %w", err)
	}

	return privateKey, publicKey, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return fallback
}
