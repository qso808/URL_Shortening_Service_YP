package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/qso808/URL_Shortening_Service_YP/internal/middleware"
	"github.com/qso808/URL_Shortening_Service_YP/internal/service"
)

// ShortenRequest представляет структуру JSON запроса для сокращения URL
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет структуру JSON ответа с сокращенным URL
type ShortenResponse struct {
	Result string `json:"result"`
}

// BatchShortenRequest представляет структуру одного элемента в batch запросе
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет структуру одного элемента в batch ответе
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse — элемент ответа GET /api/user/urls
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

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
	userID := middleware.GetUserID(r.Context())

	shortID, err := h.service.ShortenURL(longURL, userID)
	if err != nil {
		// Проверяем, является ли это ошибкой дубликата URL
		var dupErr *service.ErrDuplicateURL
		if errors.As(err, &dupErr) {
			// Формируем короткий URL из существующего shortID
			shortURL := h.baseURL + "/" + dupErr.ShortID
			
			// Устанавливаем заголовки и возвращаем ответ с кодом 409 Conflict
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
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

// ShortenURLJSON обрабатывает POST запрос /api/shorten для сокращения URL с JSON форматом
func (h *ShortenerHandler) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим JSON запрос
	var req ShortenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	userID := middleware.GetUserID(r.Context())

	shortID, err := h.service.ShortenURL(req.URL, userID)
	if err != nil {
		// Проверяем, является ли это ошибкой дубликата URL
		var dupErr *service.ErrDuplicateURL
		if errors.As(err, &dupErr) {
			// Формируем короткий URL из существующего shortID
			shortURL := h.baseURL + "/" + dupErr.ShortID

			// Создаем JSON ответ
			response := ShortenResponse{
				Result: shortURL,
			}

			// Устанавливаем заголовки
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)

			// Кодируем и отправляем JSON ответ
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Формируем короткий URL
	shortURL := h.baseURL + "/" + shortID

	// Создаем JSON ответ
	response := ShortenResponse{
		Result: shortURL,
	}

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Кодируем и отправляем JSON ответ
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// ShortenURLBatch обрабатывает POST запрос /api/shorten/batch для пакетного сокращения URL
func (h *ShortenerHandler) ShortenURLBatch(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим JSON запрос
	var requests []BatchShortenRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Проверяем, что batch не пустой
	if len(requests) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	// Преобразуем запросы в map для сервиса
	urlsMap := make(map[string]string, len(requests))
	for _, req := range requests {
		if req.CorrelationID == "" {
			http.Error(w, "correlation_id is required", http.StatusBadRequest)
			return
		}
		urlsMap[req.CorrelationID] = req.OriginalURL
	}
	userID := middleware.GetUserID(r.Context())

	results, err := h.service.ShortenURLBatch(urlsMap, userID)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Формируем ответ в том же порядке, что и запрос
	responses := make([]BatchShortenResponse, 0, len(requests))
	for _, req := range requests {
		shortID, exists := results[req.CorrelationID]
		if !exists {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		shortURL := h.baseURL + "/" + shortID
		responses = append(responses, BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// GetUserURLs обрабатывает GET /api/user/urls — возвращает все URL, сокращённые аутентифицированным пользователем.
// 401 — если cookie присутствует, но не содержит валидный user ID.
// 204 — если у пользователя нет сокращённых URL.
// 200 — JSON-массив объектов { "short_url": "http://...", "original_url": "http://..." }.
func (h *ShortenerHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	urls, err := h.service.GetUserURLs(userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response := make([]UserURLResponse, 0, len(urls))
	for _, u := range urls {
		response = append(response, UserURLResponse{
			ShortURL:    h.baseURL + "/" + u.ShortURL,
			OriginalURL: u.OriginalURL,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}