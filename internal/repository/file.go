package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/qso808/URL_Shortening_Service_YP/internal/model"
)

// FileRepository - файловая реализация репозитория
type FileRepository struct {
	mu       sync.RWMutex
	filePath string
	store    map[string]string
}

// NewFileRepository создает новый экземпляр файлового репозитория
// Загружает данные из файла при инициализации
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath: filePath,
		store:    make(map[string]string),
	}

	// Загружаем данные из файла, если он существует
	if err := repo.loadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load data from file: %w", err)
	}

	return repo, nil
}

// Save сохраняет связь между коротким ID и оригинальным URL
func (r *FileRepository) Save(id string, originalURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[id] = originalURL

	// Сохраняем в файл
	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}

	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *FileRepository) Get(id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.store[id]
	if !exists {
		return "", errors.New("URL not found")
	}

	return url, nil
}

// SaveBatch сохраняет множество URL одним разом
func (r *FileRepository) SaveBatch(mappings map[string]string) error {
	if len(mappings) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Добавляем все URL в хранилище
	for shortID, originalURL := range mappings {
		r.store[shortID] = originalURL
	}

	// Сохраняем в файл один раз
	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}

	return nil
}

// loadFromFile загружает данные из файла
func (r *FileRepository) loadFromFile() error {
	// Проверяем, существует ли файл
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		// Файл не существует - это нормально, начинаем с пустого хранилища
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Если файл пустой, начинаем с пустого хранилища
	if len(data) == 0 {
		return nil
	}

	// Парсим JSON
	var records []model.StorageRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Загружаем данные в память
	for _, record := range records {
		r.store[record.ShortURL] = record.OriginalURL
	}

	return nil
}

// saveToFile сохраняет данные в файл
func (r *FileRepository) saveToFile() error {
	// Создаем массив записей
	records := make([]model.StorageRecord, 0, len(r.store))
	for shortURL, originalURL := range r.store {
		records = append(records, model.StorageRecord{
			UUID:        uuid.New().String(),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	// Кодируем в JSON
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Создаем директорию, если она не существует
	dir := filepath.Dir(r.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Записываем в файл
	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

