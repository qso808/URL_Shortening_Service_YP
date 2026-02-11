package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/qso808/URL_Shortening_Service_YP/internal/model"
)

type fileEntry struct {
	originalURL string
	userID      string
	deleted     bool
}

// FileRepository - файловая реализация репозитория
type FileRepository struct {
	mu       sync.RWMutex
	filePath string
	store    map[string]fileEntry
}

// NewFileRepository создает новый экземпляр файлового репозитория
// Загружает данные из файла при инициализации
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath: filePath,
		store:    make(map[string]fileEntry),
	}

	// Загружаем данные из файла, если он существует
	if err := repo.loadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load data from file: %w", err)
	}

	return repo, nil
}

// Save сохраняет связь между коротким ID, оригинальным URL и user_id
func (r *FileRepository) Save(id string, originalURL string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[id] = fileEntry{originalURL: originalURL, userID: userID, deleted: false}

	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}
	return nil
}

// Get возвращает оригинальный URL по короткому ID. ErrNotFound / ErrDeleted — для выбора status code в хендлере.
func (r *FileRepository) Get(id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.store[id]
	if !exists {
		return "", ErrNotFound
	}
	if entry.deleted {
		return "", ErrDeleted
	}
	return entry.originalURL, nil
}

// GetByUserID возвращает все не удалённые URL, сокращённые пользователем userID
func (r *FileRepository) GetByUserID(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []UserURL
	for shortID, entry := range r.store {
		if entry.userID == userID && !entry.deleted {
			result = append(result, UserURL{ShortURL: shortID, OriginalURL: entry.originalURL})
		}
	}
	return result, nil
}

// MarkDeleted помечает указанные short_url как удалённые только для записей, принадлежащих userID
func (r *FileRepository) MarkDeleted(userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range shortIDs {
		if entry, ok := r.store[id]; ok && entry.userID == userID {
			entry.deleted = true
			r.store[id] = entry
		}
	}
	if err := r.saveToFile(); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}
	return nil
}

// SaveBatch сохраняет множество URL одним разом (userID для всех записей)
func (r *FileRepository) SaveBatch(mappings map[string]string, userID string) error {
	if len(mappings) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for shortID, originalURL := range mappings {
		r.store[shortID] = fileEntry{originalURL: originalURL, userID: userID, deleted: false}
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

	for _, record := range records {
		r.store[record.ShortURL] = fileEntry{
			originalURL: record.OriginalURL,
			userID:      record.UserID,
			deleted:     record.IsDeleted,
		}
	}

	return nil
}

// saveToFile сохраняет данные в файл
func (r *FileRepository) saveToFile() error {
	records := make([]model.StorageRecord, 0, len(r.store))
	for shortURL, entry := range r.store {
		records = append(records, model.StorageRecord{
			UUID:        uuid.New().String(),
			ShortURL:    shortURL,
			OriginalURL: entry.originalURL,
			UserID:      entry.userID,
			IsDeleted:   entry.deleted,
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
