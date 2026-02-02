package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qso808/URL_Shortening_Service_YP/internal/middleware"
	"github.com/qso808/URL_Shortening_Service_YP/internal/repository"
)

// mockShortenerService - мок для тестирования handlers
type mockShortenerService struct {
	shortenURLErr        error
	shortenURLResult     string
	getOriginalURLErr    error
	getOriginalURLResult string
	getUserURLsResult    []repository.UserURL
	getUserURLsErr       error
}

func (m *mockShortenerService) ShortenURL(longURL string, userID string) (string, error) {
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

func (m *mockShortenerService) ShortenURLBatch(urls map[string]string, userID string) (map[string]string, error) {
	if m.shortenURLErr != nil {
		return nil, m.shortenURLErr
	}
	result := make(map[string]string, len(urls))
	for cid := range urls {
		result[cid] = "batch_" + cid
	}
	return result, nil
}

func (m *mockShortenerService) GetUserURLs(userID string) ([]repository.UserURL, error) {
	if m.getUserURLsErr != nil {
		return nil, m.getUserURLsErr
	}
	return m.getUserURLsResult, nil
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

func TestShortenerHandler_ShortenURLJSON(t *testing.T) {
	baseURL := "http://localhost:8080"

	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		mockService    *mockShortenerService
		expectedStatus int
		expectedBody   *ShortenResponse
		expectedHeader string
	}{
		{
			name:        "успешное сокращение URL через JSON",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			mockService: &mockShortenerService{
				shortenURLResult: "EwHXdJfB",
			},
			expectedStatus: http.StatusCreated,
			expectedBody: &ShortenResponse{
				Result: "http://localhost:8080/EwHXdJfB",
			},
			expectedHeader: "application/json",
		},
		{
			name:           "неправильный метод - GET",
			method:         http.MethodGet,
			contentType:    "application/json",
			body:           `{"url":"https://practicum.yandex.ru"}`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "неправильный Content-Type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           `{"url":"https://practicum.yandex.ru"}`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "невалидный JSON",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"url":}`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "пустой URL в JSON",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"url":""}`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "отсутствует поле url в JSON",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{}`,
			mockService: &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "ошибка валидации URL в сервисе",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"invalid-url"}`,
			mockService: &mockShortenerService{
				shortenURLErr: errors.New("invalid URL"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "ошибка сохранения в репозитории",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			mockService: &mockShortenerService{
				shortenURLErr: errors.New("save error"),
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
			req := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем handler
			handler.ShortenURLJSON(rr, req)

			// Проверяем статус код
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Проверяем тело ответа для успешного случая
			if tt.expectedStatus == http.StatusCreated {
				// Проверяем Content-Type
				if rr.Header().Get("Content-Type") != tt.expectedHeader {
					t.Errorf("expected Content-Type %q, got %q", tt.expectedHeader, rr.Header().Get("Content-Type"))
				}

				// Парсим JSON ответ
				var response ShortenResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode JSON response: %v", err)
					return
				}

				// Проверяем результат
				if response.Result != tt.expectedBody.Result {
					t.Errorf("expected result %q, got %q", tt.expectedBody.Result, response.Result)
				}
			}
		})
	}
}

func TestShortenerHandler_GetUserURLs(t *testing.T) {
	baseURL := "http://localhost:8080"

	tests := []struct {
		name           string
		method         string
		userIDInCtx    string
		mockService    *mockShortenerService
		expectedStatus int
		expectedBody   []UserURLResponse
	}{
		{
			name:        "401 - нет user ID в контексте",
			method:      http.MethodGet,
			userIDInCtx: "",
			mockService: &mockShortenerService{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "204 - у пользователя нет URL",
			method:      http.MethodGet,
			userIDInCtx: "user-1",
			mockService: &mockShortenerService{
				getUserURLsResult: nil,
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:        "200 - список URL пользователя",
			method:      http.MethodGet,
			userIDInCtx: "user-1",
			mockService: &mockShortenerService{
				getUserURLsResult: []repository.UserURL{
					{ShortURL: "abc", OriginalURL: "https://example.com/1"},
					{ShortURL: "def", OriginalURL: "https://example.com/2"},
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody: []UserURLResponse{
				{ShortURL: baseURL + "/abc", OriginalURL: "https://example.com/1"},
				{ShortURL: baseURL + "/def", OriginalURL: "https://example.com/2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ShortenerHandler{service: tt.mockService, baseURL: baseURL}
			req := httptest.NewRequest(tt.method, "/api/user/urls", nil)
			if tt.userIDInCtx != "" {
				req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDContextKey, tt.userIDInCtx))
			}
			rr := httptest.NewRecorder()
			h.GetUserURLs(rr, req)
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.expectedBody != nil {
				var body []UserURLResponse
				if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
					t.Errorf("decode response: %v", err)
					return
				}
				if len(body) != len(tt.expectedBody) {
					t.Errorf("expected %d items, got %d", len(tt.expectedBody), len(body))
					return
				}
				for i := range body {
					if body[i].ShortURL != tt.expectedBody[i].ShortURL || body[i].OriginalURL != tt.expectedBody[i].OriginalURL {
						t.Errorf("item %d: expected %+v, got %+v", i, tt.expectedBody[i], body[i])
					}
				}
			}
		})
	}
}

