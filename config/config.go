package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	// API Configuration
	MovieAPIKey     string
	MovieAPIBaseURL string

	// Database Configuration
	MongoURI string
	DBName   string

	// Security Configuration
	JWTSecret string

	// Server Configuration
	Port string
	Env  string
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load environment file based on GO_ENV
	env := getEnvOrDefault("GO_ENV", "development")
	envFile := filepath.Join("environments", fmt.Sprintf(".env.%s", env))
	
	if err := godotenv.Load(envFile); err != nil {
		return nil, fmt.Errorf("error loading env file %s: %v", envFile, err)
	}

	return &Config{
		// API Configuration
		MovieAPIKey:     getEnvOrDefault("MOVIE_API_KEY", ""),
		MovieAPIBaseURL: getEnvOrDefault("MOVIE_API_BASE_URL", ""),

		// Database Configuration
		MongoURI: getEnvOrDefault("MONGO_URI", ""),
		DBName:   getEnvOrDefault("DB_NAME", "movieVsdb"),

		// Security Configuration
		JWTSecret: getEnvOrDefault("JWT_SECRET", ""),

		// Server Configuration
		Port: getEnvOrDefault("PORT", "8080"),
		Env:  env,
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
