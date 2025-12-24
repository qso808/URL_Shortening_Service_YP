package repository

import (
	"errors"
	"sync"
)

// MemoryRepository - in-memory реализация репозитория
type MemoryRepository struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewMemoryRepository создает новый экземпляр in-memory репозитория
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store: make(map[string]string),
	}
}

// Save сохраняет связь между коротким ID и оригинальным URL
func (r *MemoryRepository) Save(id string, originalURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.store[id] = originalURL
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

