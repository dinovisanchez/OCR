package config

import (
	"os"

	"github.com/biaenergy/bia-commons-go/utils"
	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabaseURL  string
	GeminiAPIKey string
}
var (
	MsName                   = utils.GetEnv("MS_NAME", "bia-ia-extraction")
	DbUser                   = utils.GetEnv("DbUser", "postgres")
	DbPass                   = utils.GetEnv("DbPass", "")
	DbHost                   = utils.GetEnv("DbHost", "localhost")
	DbPort                   = utils.GetEnv("DbPort", "5432")
	DbName                   = utils.GetEnv("DbName", "bia-ia-extraction")
	DbSSLMode                = utils.GetEnv("DbSSLMode", "require")
	DbMigrationFile          = "file://.//provider//postgres//migrations"
	ProdEnvironment          = utils.GetEnv("GO_ENVIRONMENT", "develop") == "production"
)

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
} 