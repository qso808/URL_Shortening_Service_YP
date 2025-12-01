package handler

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/qso808/URL_Shortening_Service_YP/internal/service"
)

// ShortenerHandler обрабатывает HTTP запросы для сервиса сокращения URL
type ShortenerHandler struct {
	service service.Shortener
	baseURL string
}

// NewShortenerHandler создает новый экземпляр хэндлера
func NewShortenerHandler(svc service.Shortener, baseURL string) *ShortenerHandler {
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
	// Проверяем метод (для обратной совместимости с тестами)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Извлекаем ID из параметров роутера (chi)
	shortID := chi.URLParam(r, "id")
	
	// Если ID не найден в параметрах (для обратной совместимости), пробуем извлечь из пути
	if shortID == "" {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		shortID = path
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

