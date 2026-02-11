package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/qso808/URL_Shortening_Service_YP/internal/repository"
)

// ErrDuplicateURL - ошибка, возникающая при попытке сократить уже существующий URL
// Содержит существующий shortID
type ErrDuplicateURL struct {
	ShortID string
}

func (e *ErrDuplicateURL) Error() string {
	return fmt.Sprintf("URL already exists with short ID: %s", e.ShortID)
}

// ShortenerService содержит бизнес-логику для сокращения URL
type ShortenerService struct {
	repo Repository
}

// Repository определяет интерфейс репозитория
type Repository interface {
	Save(id string, originalURL string) error
	Get(id string) (string, error)
}

// Shortener определяет интерфейс сервиса сокращения URL
type Shortener interface {
	ShortenURL(longURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
	ShortenURLBatch(urls map[string]string) (map[string]string, error)
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
		// Проверяем, является ли это ошибкой дубликата URL
		if err == repository.ErrDuplicateURL {
			// Получаем существующий shortID по originalURL
			// Используем type assertion для проверки наличия метода GetByOriginalURL
			if postgresRepo, ok := s.repo.(interface {
				GetByOriginalURL(originalURL string) (string, error)
			}); ok {
				existingShortID, getErr := postgresRepo.GetByOriginalURL(longURL)
				if getErr != nil {
					return "", fmt.Errorf("failed to get existing URL: %w", getErr)
				}
				return "", &ErrDuplicateURL{ShortID: existingShortID}
			}
			// Если репозиторий не поддерживает GetByOriginalURL, возвращаем общую ошибку
			return "", err
		}
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

// ShortenURLBatch создает короткие идентификаторы для множества URL
// Принимает map[correlationID]originalURL и возвращает map[correlationID]shortID
func (s *ShortenerService) ShortenURLBatch(urls map[string]string) (map[string]string, error) {
	if len(urls) == 0 {
		return nil, errors.New("empty batch")
	}

	result := make(map[string]string, len(urls))

	// Валидируем и генерируем ID для всех URL
	type urlData struct {
		correlationID string
		originalURL   string
		shortID       string
	}

	urlsToSave := make([]urlData, 0, len(urls))
	for correlationID, originalURL := range urls {
		// Валидация URL
		if err := s.validateURL(originalURL); err != nil {
			return nil, fmt.Errorf("invalid URL for correlation_id %s: %w", correlationID, err)
		}

		// Генерируем уникальный короткий ID
		shortID := s.generateShortID()
		result[correlationID] = shortID

		urlsToSave = append(urlsToSave, urlData{
			correlationID: correlationID,
			originalURL:   originalURL,
			shortID:       shortID,
		})
	}

	// Сохраняем все URL в репозиторий
	// Для PostgreSQL используем batch сохранение в транзакции
	// Для других репозиториев используем обычный Save
	if batchRepo, ok := s.repo.(interface {
		SaveBatch(mappings map[string]string) error
	}); ok {
		// Используем batch сохранение
		batchMappings := make(map[string]string, len(urlsToSave))
		for _, data := range urlsToSave {
			batchMappings[data.shortID] = data.originalURL
		}
		if err := batchRepo.SaveBatch(batchMappings); err != nil {
			return nil, fmt.Errorf("failed to save batch: %w", err)
		}
	} else {
		// Используем обычный Save для каждого URL
		for _, data := range urlsToSave {
			if err := s.repo.Save(data.shortID, data.originalURL); err != nil {
				return nil, fmt.Errorf("failed to save URL for correlation_id %s: %w", data.correlationID, err)
			}
		}
	}

	return result, nil
}