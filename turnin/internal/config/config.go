package config

import (
	"errors"
	"log"
	"os"
	"time"
)

type Config struct {
	Env            string
	Addr           string
	DBPath         string
	JWTSecret      string
	JWTTTL         time.Duration
	UploadDir      string
	AllowedOrigins string
	ScanDuration   time.Duration
	MaxUploadBytes int64
}

func Load() (*Config, error) {
	scan, err := time.ParseDuration(env("SCAN_DURATION", "5s"))
	if err != nil {
		return nil, errors.New("SCAN_DURATION must be a valid Go duration, e.g. 5s")
	}

	cfg := &Config{
		Env:            env("APP_ENV", "development"),
		Addr:           env("ADDR", ":8080"),
		DBPath:         env("DB_PATH", "turnin.db"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTTTL:         12 * time.Hour,
		UploadDir:      env("UPLOAD_DIR", "uploads"),
		AllowedOrigins: env("CORS_ORIGINS", "http://localhost:3000"),
		ScanDuration:   scan,
		MaxUploadBytes: 10 << 20,
	}

	if cfg.JWTSecret == "" {
		if cfg.Env == "production" {
			return nil, errors.New("JWT_SECRET is required when APP_ENV=production")
		}
		cfg.JWTSecret = "dev-only-insecure-secret-change-me"
		log.Println("[config] WARNING: using insecure development JWT secret")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
