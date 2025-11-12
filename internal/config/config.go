package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Env      string         `mapstructure:"env"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	// Читаем config.yaml с параметрами по умолчанию
	configPath := getEnv("CONFIG_PATH", "config.yaml")
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Читаем YAML (игнорируем ошибку, если файла нет)
	_ = viper.ReadInConfig()

	// Настраиваем чтение переменных окружения
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Явный биндинг для переменных окружения (для docker-compose)
	bindEnvVariables()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Переопределяем из ENV (приоритет над YAML)
	overrideFromEnv(&cfg)
	
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func bindEnvVariables() {
	// Биндим переменные окружения к путям в структуре
	_ = viper.BindEnv("database.host", "APP_DATABASE_HOST")
	_ = viper.BindEnv("database.port", "APP_DATABASE_PORT")
	_ = viper.BindEnv("database.user", "APP_DATABASE_USER")
	_ = viper.BindEnv("database.password", "APP_DATABASE_PASSWORD")
	_ = viper.BindEnv("database.name", "APP_DATABASE_NAME")
	_ = viper.BindEnv("server.port", "APP_SERVER_PORT")
	_ = viper.BindEnv("server.host", "APP_SERVER_HOST")
}

func overrideFromEnv(cfg *Config) {
	// Явное чтение критичных ENV
	if v := os.Getenv("APP_DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("APP_DATABASE_PORT"); v != "" {
		cfg.Database.Port = v
	}
	if v := os.Getenv("APP_DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("APP_DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("APP_DATABASE_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("APP_SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("APP_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
}

func validate(cfg *Config) error {
	if cfg.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

// GetDSN возвращает connection string для PostgreSQL
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
