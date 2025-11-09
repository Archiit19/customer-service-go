package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort string

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

func mustAtoi32(key string, def int32) int32 {
	v := getenv(key, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("WARN: invalid %s=%q; using default %d", key, v, def)
		return def
	}
	return int32(i)
}

func mustParseDuration(key string, def time.Duration) time.Duration {
	v := getenv(key, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("WARN: invalid %s=%q; using default %s", key, v, def)
		return def
	}
	return d
}

func Load() *Config {
	return &Config{
		AppPort: getenv("APP_PORT", "8080"),

		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USER", "postgres"),
		DBPassword: getenv("DB_PASSWORD", "postgres"),
		DBName:     getenv("DB_NAME", "postgres"),
		DBSSLMode:  getenv("DB_SSLMODE", "disable"), // use "require" for AWS RDS

		DBMaxConns:    mustAtoi32("DB_MAX_CONNS", 10),
		DBMinConns:    mustAtoi32("DB_MIN_CONNS", 2),
		DBMaxIdleTime: mustParseDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
	}
}
