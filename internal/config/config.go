package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	Host          string
	DatabasePath  string
	AdminEmail    string
	AdminPassword string
	SeedData      bool
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	host := getEnv("HOST", "0.0.0.0")
	dbPath := getEnv("DATABASE_PATH", "./data/pocketa.db")
	adminEmail := getEnv("ADMIN_EMAIL", "risk@pocketa.money")
	adminPass := getEnv("ADMIN_PASSWORD", "pocketa-admin")
	seedData := getEnvBool("SEED_DATA", true)

	return &Config{
		Port:          port,
		Host:          host,
		DatabasePath:  dbPath,
		AdminEmail:    adminEmail,
		AdminPassword: adminPass,
		SeedData:      seedData,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return defaultVal
}
