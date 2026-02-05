package repository

import (
	"errors"
	"sync"
)

type memoryEntry struct {
	originalURL string
	userID      string
	deleted     bool
}

// MemoryRepository - in-memory реализация репозитория
type MemoryRepository struct {
	mu       sync.RWMutex
	store    map[string]memoryEntry // shortID -> entry
	userURLs map[string][]string    // userID -> []shortID
}

// NewMemoryRepository создает новый экземпляр in-memory репозитория
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store:    make(map[string]memoryEntry),
		userURLs: make(map[string][]string),
	}
}

// Save сохраняет связь между коротким ID, оригинальным URL и user_id
func (r *MemoryRepository) Save(id string, originalURL string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[id] = memoryEntry{originalURL: originalURL, userID: userID}
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

// Get возвращает оригинальный URL по короткому ID и флаг удаления
func (r *MemoryRepository) Get(id string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.store[id]
	if !exists {
		return "", false, errors.New("URL not found")
	}
	return entry.originalURL, entry.deleted, nil
}

// GetByUserID возвращает все не удалённые URL, сокращённые пользователем userID
func (r *MemoryRepository) GetByUserID(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.userURLs[userID]
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]UserURL, 0, len(ids))
	for _, shortID := range ids {
		entry, ok := r.store[shortID]
		if !ok || entry.deleted {
			continue
		}
		result = append(result, UserURL{ShortURL: shortID, OriginalURL: entry.originalURL})
	}
	return result, nil
}

// MarkDeleted помечает указанные short_url как удалённые только для записей, принадлежащих userID
func (r *MemoryRepository) MarkDeleted(userID string, shortIDs []string) error {
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
	return nil
}
