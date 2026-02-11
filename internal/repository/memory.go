package repository

import (
	"errors"
	"sync"
)

// MemoryRepository - in-memory реализация репозитория
type MemoryRepository struct {
	mu       sync.RWMutex
	store    map[string]string       // shortID -> originalURL
	userURLs map[string][]string     // userID -> []shortID
}

// NewMemoryRepository создает новый экземпляр in-memory репозитория
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store:    make(map[string]string),
		userURLs: make(map[string][]string),
	}
}

// Save сохраняет связь между коротким ID, оригинальным URL и user_id
func (r *MemoryRepository) Save(id string, originalURL string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[id] = originalURL
	if userID != "" {
		ids := r.userURLs[userID]
		for _, sid := range ids {
			if sid == id {
				return nil
			}
		}
		r.userURLs[userID] = append(r.userURLs[userID], id)
	}
	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *MemoryRepository) Get(id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.store[id]
	if !exists {
		return "", errors.New("URL not found")
	}
	return url, nil
}

// GetByUserID возвращает все URL, сокращённые пользователем userID
func (r *MemoryRepository) GetByUserID(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.userURLs[userID]
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]UserURL, 0, len(ids))
	for _, shortID := range ids {
		originalURL, ok := r.store[shortID]
		if !ok {
			continue
		}
		result = append(result, UserURL{ShortURL: shortID, OriginalURL: originalURL})
	}
	return result, nil
}

