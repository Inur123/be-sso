package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// App
	AppPort string
	AppEnv  string
	AppName string
	AppURL  string
	APIURL  string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// JWT
	JWTSecret        string
	JWTAccessExpire  int
	JWTRefreshExpire int

	// Encryption key for DB fields + avatar files (AES-256, 32 byte hex)
	EncryptionKey string

	// SMTP
	SMTPHost     string
	SMTPPort     int
	SMTPEmail    string
	SMTPPassword string
}

var cfg *Config

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	accessExpire, _ := strconv.Atoi(getEnv("JWT_ACCESS_EXPIRE", "3600"))
	refreshExpire, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXPIRE", "604800"))

	cfg = &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppName: getEnv("APP_NAME", "SSO IPNU-IPPNU Magetan"),
		AppURL:  getEnv("APP_URL", "http://localhost:3000"),
		APIURL:  getEnv("API_URL", "http://localhost:8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "sso_pelajarnu"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		JWTSecret:        getEnv("JWT_SECRET", "change-this-secret"),
		JWTAccessExpire:  accessExpire,
		JWTRefreshExpire: refreshExpire,

		EncryptionKey: getEnv("ENCRYPTION_KEY", "0000000000000000000000000000000000000000000000000000000000000000"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     func() int { p, _ := strconv.Atoi(getEnv("SMTP_PORT", "587")); return p }(),
		SMTPEmail:    getEnv("SMTP_EMAIL", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""), // App Password Google
	}

	return cfg
}

func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
