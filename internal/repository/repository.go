package repository

import "errors"

// Ошибки, возвращаемые Get; по ним в хендлере выбирается HTTP status code (errors.Is).
var (
	ErrNotFound = errors.New("url not found")
	ErrDeleted  = errors.New("url deleted")
)

// UserURL — пара short_url и original_url для ответа GET /api/user/urls
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// Repository определяет интерфейс для работы с хранилищем URL
type Repository interface {
	// Save сохраняет связь между коротким ID, оригинальным URL и идентификатором пользователя
	Save(id string, originalURL string, userID string) error

	// Get возвращает оригинальный URL по короткому ID. ErrNotFound — запись не найдена, ErrDeleted — soft delete.
	Get(id string) (originalURL string, err error)

	// GetByUserID возвращает все не удалённые URL, сокращённые пользователем userID
	GetByUserID(userID string) ([]UserURL, error)

	// MarkDeleted помечает URL как удалённые (только записи, принадлежащие userID)
	MarkDeleted(userID string, shortIDs []string) error
}
