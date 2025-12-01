package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockShortenerService - мок для тестирования handlers
// Реализует интерфейс service.Shortener
type mockShortenerService struct {
	shortenURLErr         error
	shortenURLResult      string
	getOriginalURLErr     error
	getOriginalURLResult  string
}

func (m *mockShortenerService) ShortenURL(longURL string) (string, error) {
	if m.shortenURLErr != nil {
		return "", m.shortenURLErr
	}
	return m.shortenURLResult, nil
}

func (m *mockShortenerService) GetOriginalURL(shortID string) (string, error) {
	if m.getOriginalURLErr != nil {
		return "", m.getOriginalURLErr
	}
	return m.getOriginalURLResult, nil
}

func TestShortenerHandler_ShortenURL(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	tests := []struct {
		name           string
		method         string
		body           string
		mockService    *mockShortenerService
		expectedStatus int
		expectedBody   string
		expectedHeader string
	}{
		{
			name:   "успешное сокращение URL",
			method: http.MethodPost,
			body:   "https://practicum.yandex.ru/",
			mockService: &mockShortenerService{
				shortenURLResult: "EwHXdJfB",
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/EwHXdJfB",
			expectedHeader: "text/plain",
		},
		{
			name:           "неправильный метод - GET",
			method:         http.MethodGet,
			body:           "https://practicum.yandex.ru/",
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "неправильный метод - PUT",
			method:         http.MethodPut,
			body:           "https://practicum.yandex.ru/",
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "ошибка валидации URL в сервисе",
			method: http.MethodPost,
			body:   "invalid-url",
			mockService: &mockShortenerService{
				shortenURLErr: errors.New("invalid URL"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "ошибка сохранения в репозитории",
			method: http.MethodPost,
			body:   "https://practicum.yandex.ru/",
			mockService: &mockShortenerService{
				shortenURLErr: errors.New("save error"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "пустое тело запроса",
			method:         http.MethodPost,
			body:           "",
			mockService: &mockShortenerService{
				shortenURLErr: errors.New("empty URL"),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем handler с мок-сервисом
			handler := &ShortenerHandler{
				service: tt.mockService,
				baseURL: baseURL,
			}

			// Создаем запрос
			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "text/plain")

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем handler
			handler.ShortenURL(rr, req)

			// Проверяем статус код
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Проверяем тело ответа для успешного случая
			if tt.expectedStatus == http.StatusCreated {
				if rr.Body.String() != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, rr.Body.String())
				}
				if rr.Header().Get("Content-Type") != tt.expectedHeader {
					t.Errorf("expected Content-Type %q, got %q", tt.expectedHeader, rr.Header().Get("Content-Type"))
				}
			}
		})
	}
}

func TestShortenerHandler_Redirect(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	tests := []struct {
		name           string
		method         string
		path           string
		mockService    *mockShortenerService
		expectedStatus int
		expectedLocation string
	}{
		{
			name:   "успешный редирект",
			method: http.MethodGet,
			path:   "/EwHXdJfB",
			mockService: &mockShortenerService{
				getOriginalURLResult: "https://practicum.yandex.ru/",
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://practicum.yandex.ru/",
		},
		{
			name:   "успешный редирект с длинным ID",
			method: http.MethodGet,
			path:   "/AbCdEfGh",
			mockService: &mockShortenerService{
				getOriginalURLResult: "https://example.com/page",
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://example.com/page",
		},
		{
			name:           "неправильный метод - POST",
			method:         http.MethodPost,
			path:           "/EwHXdJfB",
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "неправильный метод - PUT",
			method:         http.MethodPut,
			path:           "/EwHXdJfB",
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "URL не найден",
			method: http.MethodGet,
			path:   "/NonExistentID",
			mockService: &mockShortenerService{
				getOriginalURLErr: errors.New("URL not found"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "пустой ID",
			method: http.MethodGet,
			path:   "/",
			mockService: &mockShortenerService{
				getOriginalURLErr: errors.New("empty short ID"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "ошибка получения URL из репозитория",
			method: http.MethodGet,
			path:   "/EwHXdJfB",
			mockService: &mockShortenerService{
				getOriginalURLErr: errors.New("repository error"),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем handler с мок-сервисом
			handler := &ShortenerHandler{
				service: tt.mockService,
				baseURL: baseURL,
			}

			// Создаем запрос
			req := httptest.NewRequest(tt.method, tt.path, nil)

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем handler
			handler.Redirect(rr, req)

			// Проверяем статус код
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Проверяем заголовок Location для успешного редиректа
			if tt.expectedStatus == http.StatusTemporaryRedirect {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected Location %q, got %q", tt.expectedLocation, location)
				}
			}
		})
	}
}

