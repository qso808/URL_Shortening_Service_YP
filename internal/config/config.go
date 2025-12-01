package config

import (
	"flag"
	"fmt"
	"net/url"
)

// Config содержит конфигурацию приложения
type Config struct {
	// ServerAddress адрес запуска HTTP-сервера (например, localhost:8080)
	ServerAddress string
	
	// BaseURL базовый адрес результирующего сокращённого URL (например, http://localhost:8080)
	BaseURL string
}

// LoadConfig загружает конфигурацию из аргументов командной строки
// Возвращает ошибку, если конфигурация некорректна
func LoadConfig() (*Config, error) {
	var serverAddr string
	var baseURL string
	
	// Определяем флаги командной строки
	flag.StringVar(&serverAddr, "a", "localhost:8080", "адрес запуска HTTP-сервера")
	flag.StringVar(&baseURL, "b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	
	// Парсим аргументы командной строки
	flag.Parse()
	
	// Создаем конфигурацию
	cfg := &Config{
		ServerAddress: serverAddr,
		BaseURL:       baseURL,
	}
	
	// Валидируем конфигурацию
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}
	
	// Проверяем, что BaseURL является валидным URL
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL format: %w", err)
	}
	
	// URL должен иметь схему (http или https)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("base URL must have http or https scheme")
	}
	
	// URL должен иметь host
	if parsedURL.Host == "" {
		return fmt.Errorf("base URL must have a host")
	}
	
	return nil
}

