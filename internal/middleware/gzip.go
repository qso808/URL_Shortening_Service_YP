package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware создает middleware для поддержки gzip компрессии
// Декомпрессирует входящие запросы с Content-Encoding: gzip
// Компрессирует исходящие ответы для клиентов с Accept-Encoding: gzip
// Работает только для application/json и text/html
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Декомпрессия входящего запроса
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Bad Request: invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = io.NopCloser(gzReader)
		}

		// Проверяем, поддерживает ли клиент gzip
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		// Создаем обертку для ResponseWriter
		writer := &gzipResponseWriter{
			ResponseWriter: w,
			acceptsGzip:    acceptsGzip,
		}

		// Выполняем следующий обработчик
		next.ServeHTTP(writer, r)

		// Закрываем gzip writer, если он был создан
		if writer.gzipWriter != nil {
			writer.gzipWriter.Close()
		}
	})
}

// gzipResponseWriter обертка над http.ResponseWriter для компрессии ответов
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter  *gzip.Writer
	acceptsGzip bool
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Если gzip writer еще не создан, проверяем Content-Type
	if w.gzipWriter == nil {
		// Получаем Content-Type из заголовков
		contentType := w.Header().Get("Content-Type")

		// Проверяем, нужно ли сжимать ответ
		shouldCompress := w.acceptsGzip &&
			(strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "text/html"))

		if shouldCompress {
			// Создаем gzip writer
			w.gzipWriter = gzip.NewWriter(w.ResponseWriter)
			w.Header().Set("Content-Encoding", "gzip")
			// Удаляем Content-Length, так как размер изменится после сжатия
			w.Header().Del("Content-Length")
		}
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	// Если заголовки еще не записаны, проверяем Content-Type и решаем, нужно ли сжимать
	if !w.wroteHeader {
		// Получаем Content-Type из заголовков
		contentType := w.Header().Get("Content-Type")

		// Проверяем, нужно ли сжимать ответ
		shouldCompress := w.acceptsGzip &&
			(strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "text/html"))

		if shouldCompress {
			// Создаем gzip writer
			w.gzipWriter = gzip.NewWriter(w.ResponseWriter)
			w.Header().Set("Content-Encoding", "gzip")
			// Удаляем Content-Length, так как размер изменится после сжатия
			w.Header().Del("Content-Length")
		}

		// Вызываем WriteHeader
		w.WriteHeader(http.StatusOK)
	}

	// Если gzip writer создан, пишем через него
	if w.gzipWriter != nil {
		return w.gzipWriter.Write(b)
	}
	// Иначе пишем напрямую
	return w.ResponseWriter.Write(b)
}

