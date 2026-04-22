package config

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    JWTSecret  string
}

var AppConfig *Config

func LoadConfig() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Println("Warning: .env file not found, using environment variables")
    }

    AppConfig = &Config{
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBPort:     getEnv("DB_PORT", "3306"),
        DBUser:     getEnv("DB_USER", "root"),
        DBPassword: getEnv("DB_PASSWORD", ""),
        DBName:     getEnv("DB_NAME", "money_manager"),
        JWTSecret:  getEnv("JWT_SECRET", "your-secret-key"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}