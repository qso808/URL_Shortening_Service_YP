package model

// StorageRecord представляет запись в файловом хранилище
type StorageRecord struct {
	UUID       string `json:"uuid"`
	ShortURL   string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

