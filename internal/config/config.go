package config

import (
	"os"
)

type Config struct {
	Port       string
	DBConnStr  string
	StorageDir string
}

func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBConnStr:  getEnv("DB_CONN_STR", "postgres://postgres:postgres@localhost:5432/objectvault?sslmode=disable"),
		StorageDir: getEnv("STORAGE_DIR", "./storage"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
