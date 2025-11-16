package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig   // Настройки HTTP сервера
	Database DatabaseConfig // Настройки подключения к БД
	JWT      JWTConfig      // Настройки JWT авторизации
}

// ServerConfig содержит настройки HTTP сервера
type ServerConfig struct {
	Port string `envconfig:"SERVER_PORT" default:"8080"`
	Host string `envconfig:"SERVER_HOST" default:"0.0.0.0"`
}

// DatabaseConfig содержит настройки подключения к PostgreSQL
type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST" default:"localhost"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" default:"pr_service"`
	Password string `envconfig:"DB_PASSWORD" default:"pr_service_pass"`
	Name     string `envconfig:"DB_NAME" default:"pr_service"`
	SSLMode  string `envconfig:"DB_SSLMODE" default:"disable"`
	MaxConns int32  `envconfig:"DB_MAX_CONNS" default:"25"`
	MinConns int32  `envconfig:"DB_MIN_CONNS" default:"5"`
}

// JWTConfig содержит настройки JWT авторизации
type JWTConfig struct {
	Secret          string `envconfig:"JWT_SECRET" required:"true"`
	ExpirationHours int    `envconfig:"JWT_EXPIRATION_HOURS" default:"24"`
}

// GetExpiration возвращает срок действия токена как time.Duration
func (j JWTConfig) GetExpiration() time.Duration {
	return time.Duration(j.ExpirationHours) * time.Hour
}

// DSN возвращает строку подключения к PostgreSQL
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// Load читает конфигурацию из переменных окружения
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
