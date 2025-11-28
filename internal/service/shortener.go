package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
)

// ShortenerService содержит бизнес-логику для сокращения URL
type ShortenerService struct {
	repo Repository
}

// Repository определяет интерфейс репозитория
type Repository interface {
	Save(id string, originalURL string) error
	Get(id string) (string, error)
}

// NewShortenerService создает новый экземпляр сервиса
func NewShortenerService(repo Repository) *ShortenerService {
	return &ShortenerService{
		repo: repo,
	}
}

// ShortenURL создает короткий идентификатор для длинного URL
func (s *ShortenerService) ShortenURL(longURL string) (string, error) {
	// Валидация URL
	if err := s.validateURL(longURL); err != nil {
		return "", err
	}
	
	// Генерируем уникальный короткий ID
	shortID := s.generateShortID()
	
	// Сохраняем в репозиторий
	if err := s.repo.Save(shortID, longURL); err != nil {
		return "", err
	}
	
	return shortID, nil
}

// GetOriginalURL возвращает оригинальный URL по короткому ID
func (s *ShortenerService) GetOriginalURL(shortID string) (string, error) {
	if shortID == "" {
		return "", errors.New("empty short ID")
	}
	
	return s.repo.Get(shortID)
}

// validateURL проверяет корректность URL
func (s *ShortenerService) validateURL(urlStr string) error {
	if urlStr == "" {
		return errors.New("empty URL")
	}
	
	// Проверяем, что это валидный URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.New("invalid URL format")
	}
	
	// URL должен иметь схему (http или https)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must have http or https scheme")
	}
	
	// URL должен иметь host
	if parsedURL.Host == "" {
		return errors.New("URL must have a host")
	}
	
	return nil
}

// generateShortID генерирует короткий уникальный идентификатор
func (s *ShortenerService) generateShortID() string {
	// Генерируем 6 байт случайных данных
	b := make([]byte, 6)
	rand.Read(b)
	
	// Кодируем в base64 URL-safe формат и берем первые 8 символов
	encoded := base64.URLEncoding.EncodeToString(b)
	// Убираем padding и берем первые 8 символов
	encoded = strings.TrimRight(encoded, "=")
	if len(encoded) > 8 {
		encoded = encoded[:8]
	}
	
	return encoded
}

