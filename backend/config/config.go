package config

import (
    "log"
    "os"
	"strings"
    "github.com/joho/godotenv"
)

type Config struct {
    DBHost     		string
    DBPort     		string
    DBUser     		string
    DBPassword 		string
    DBName     		string
    JWTSecret  		string
	ServerPort     	string
    CorsOrigins    	[]string
}

var AppConfig *Config

func LoadConfig() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Println("Warning: .env file not found, using environment variables")
    }

	// Parse CORS origins from comma-separated string
    corsOriginsStr := getEnv("CORS_ORIGINS", "http://localhost:3000")
    corsOrigins := strings.Split(corsOriginsStr, ",")
    
    // Trim spaces
    for i, origin := range corsOrigins {
        corsOrigins[i] = strings.TrimSpace(origin)
    }

    AppConfig = &Config{
        DBHost:     	getEnv("DB_HOST", "localhost"),
        DBPort:     	getEnv("DB_PORT", "3306"),
        DBUser:     	getEnv("DB_USER", "root"),
        DBPassword: 	getEnv("DB_PASSWORD", ""),
        DBName:     	getEnv("DB_NAME", "money_manager"),
        JWTSecret:  	getEnv("JWT_SECRET", "your-secret-key"),
		ServerPort:    	getEnv("SERVER_PORT", "5000"),
        CorsOrigins:    corsOrigins,
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}