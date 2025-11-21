package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	ClerkSecretKey  string
	OPADir          string
	AllowedOrigins  string
	Port            string
	AWSRegion       string
	AzureTenantID   string
	GCPProjectID    string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/finopsbridge?sslmode=disable"),
		ClerkSecretKey: getEnv("CLERK_SECRET_KEY", ""),
		OPADir:         getEnv("OPA_DIR", "./policies"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		Port:           getEnv("PORT", "8080"),
		AWSRegion:      getEnv("AWS_REGION", "us-east-1"),
		AzureTenantID:   getEnv("AZURE_TENANT_ID", ""),
		GCPProjectID:    getEnv("GCP_PROJECT_ID", ""),
	}
}

func (c *Config) GetAllowedOrigins() []string {
	return strings.Split(c.AllowedOrigins, ",")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

