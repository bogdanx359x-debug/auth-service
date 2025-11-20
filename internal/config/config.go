package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	TokenTTL        time.Duration
	ShutdownTimeout time.Duration
}

func Load() Config {
	dbURL := mustEnv("DB_DSN")
	jwtSecret := mustEnv("JWT_SECRET")

	port := getenvDefault("PORT", "8080")
	tokenTTL := durationEnv("TOKEN_TTL_MIN", 60) * time.Minute
	shutdown := durationEnv("SHUTDOWN_TIMEOUT_SEC", 5) * time.Second

	return Config{
		Port:            port,
		DatabaseURL:     dbURL,
		JWTSecret:       jwtSecret,
		TokenTTL:        tokenTTL,
		ShutdownTimeout: shutdown,
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return val
}

func getenvDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func durationEnv(key string, def int) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return time.Duration(def)
	}
	i, err := strconv.Atoi(val)
	if err != nil || i <= 0 {
		log.Printf("invalid %s, using default %d", key, def)
		return time.Duration(def)
	}
	return time.Duration(i)
}
