package handler

import (
	"io"
	"net/http"

	"github.com/qso808/URL_Shortening_Service_YP/internal/service"
)

// ShortenerHandler обрабатывает HTTP запросы для сервиса сокращения URL
type ShortenerHandler struct {
	service *service.ShortenerService
	baseURL string
}

// NewShortenerHandler создает новый экземпляр хэндлера
func NewShortenerHandler(svc *service.ShortenerService, baseURL string) *ShortenerHandler {
	return &ShortenerHandler{
		service: svc,
		baseURL: baseURL,
	}
}

// ShortenURL обрабатывает POST запрос для сокращения URL
func (h *ShortenerHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	longURL := string(body)
	
	// Сокращаем URL через сервис
	shortID, err := h.service.ShortenURL(longURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	// Формируем короткий URL
	shortURL := h.baseURL + "/" + shortID
	
	// Устанавливаем заголовки и возвращаем ответ
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// Redirect обрабатывает GET запрос для редиректа на оригинальный URL
func (h *ShortenerHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Извлекаем ID из пути (убираем ведущий слэш)
	shortID := r.URL.Path
	if len(shortID) > 0 && shortID[0] == '/' {
		shortID = shortID[1:]
	}
	
	// Получаем оригинальный URL
	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	// Выполняем редирект
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

