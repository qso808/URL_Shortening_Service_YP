package repository

// Repository определяет интерфейс для работы с хранилищем URL
type Repository interface {
	// Save сохраняет связь между коротким ID и оригинальным URL
	Save(id string, originalURL string) error
	
	// Get возвращает оригинальный URL по короткому ID
	Get(id string) (string, error)
}

