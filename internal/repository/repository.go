package repository

// UserURL — пара short_url и original_url для ответа GET /api/user/urls
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// Repository определяет интерфейс для работы с хранилищем URL
type Repository interface {
	// Save сохраняет связь между коротким ID, оригинальным URL и идентификатором пользователя
	Save(id string, originalURL string, userID string) error

	// Get возвращает оригинальный URL по короткому ID
	Get(id string) (string, error)

	// GetByUserID возвращает все URL, сокращённые пользователем userID
	GetByUserID(userID string) ([]UserURL, error)
}

