package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port            string
	ShutdownTimeout int
}

type LoggerConfig struct {
	Level string
}

type DatabaseConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type PoolConfig struct {
	MaxConns    int
	MinConns    int
	MaxIdleTime time.Duration
	MaxLifeTime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Host  string
	Port  string
	Topic string
}

type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Pool     PoolConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	Mail     MailConfig
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

func New() (*Config, error) {
	err := godotenv.Load("/.env")
	if err != nil {
		return nil, fmt.Errorf("failed to load env: %w", err)
	}
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			ShutdownTimeout: getEnvAsInt("SERVER_PORT", 15),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "redis"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "postgres"),
			Port:     getEnv("DB_PORT", "5435"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "postgres"),
		},
		Pool: PoolConfig{
			MaxConns:    getEnvAsInt("POOL_MAX_CONNS", 10),
			MinConns:    getEnvAsInt("POOL_MIN_CONNS", 2),
			MaxIdleTime: getEnvAsDuration("POOL_MAX_IDLE_TIME", time.Hour),
			MaxLifeTime: getEnvAsDuration("POOL_MAX_LIFE_TIME", 10*time.Minute),
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "debug"),
		},
		Kafka: KafkaConfig{
			Host:  getEnv("KAFKA_HOST", "kafka"),
			Port:  getEnv("KAFKA_PORT", "9092"),
			Topic: getEnv("KAFKA_TOPIC", "notify-topic"),
		},
		Mail: MailConfig{
			Host:     getEnv("MAIL_HOST", ""),
			Port:     getEnvAsInt("MAIL_PORT", 587),
			User:     getEnv("MAIL_USER", "notifier-app"),
			Password: getEnv("MAIL_PASSWORD", ""),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
