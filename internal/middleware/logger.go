package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// ResponseWriter обертка над http.ResponseWriter для отслеживания статуса и размера ответа
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// RequestLogger создает middleware для логирования HTTP запросов и ответов
func RequestLogger(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем обертку для ResponseWriter
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // значение по умолчанию
			}

			// Выполняем следующий обработчик
			next.ServeHTTP(wrapped, r)

			// Вычисляем время выполнения
			duration := time.Since(start)

			// Логируем информацию о запросе и ответе на уровне Info
			logger.Info().
				Str("uri", r.RequestURI).
				Str("method", r.Method).
				Int("status", wrapped.statusCode).
				Int("size", wrapped.size).
				Dur("duration", duration).
				Msg("request processed")
		})
	}
}

