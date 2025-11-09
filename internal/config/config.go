package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppPort  string
	LogLevel string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	DBMaxConns    int32
	DBMinConns    int32
	DBMaxIdleTime time.Duration
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt32(key string, def int32) (int32, string) {
	v := getenv(key, "")
	if v == "" {
		return def, ""
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Sprintf("invalid %s=%q; using default %d", key, v, def)
	}
	return int32(i), ""
}

func parseDuration(key string, def time.Duration) (time.Duration, string) {
	v := getenv(key, "")
	if v == "" {
		return def, ""
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def, fmt.Sprintf("invalid %s=%q; using default %s", key, v, def)
	}
	return d, ""
}

func Load() (*Config, []string) {
	warnings := make([]string, 0)
	maxConns, warn := parseInt32("DB_MAX_CONNS", 10)
	if warn != "" {
		warnings = append(warnings, warn)
	}
	minConns, warn := parseInt32("DB_MIN_CONNS", 2)
	if warn != "" {
		warnings = append(warnings, warn)
	}
	maxIdle, warn := parseDuration("DB_MAX_IDLE_TIME", 30*time.Minute)
	if warn != "" {
		warnings = append(warnings, warn)
	}

	cfg := &Config{
		AppPort:  getenv("APP_PORT", "8080"),
		LogLevel: strings.ToUpper(getenv("LOG_LEVEL", "INFO")),

		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USER", "postgres"),
		DBPassword: getenv("DB_PASSWORD", "postgres"),
		DBName:     getenv("DB_NAME", "postgres"),
		DBSSLMode:  getenv("DB_SSLMODE", "disable"),

		DBMaxConns:    maxConns,
		DBMinConns:    minConns,
		DBMaxIdleTime: maxIdle,
	}
	return cfg, warnings
}
