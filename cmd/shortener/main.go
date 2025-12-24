package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/qso808/URL_Shortening_Service_YP/internal/config"
	"github.com/qso808/URL_Shortening_Service_YP/internal/handler"
	customMiddleware "github.com/qso808/URL_Shortening_Service_YP/internal/middleware"
	"github.com/qso808/URL_Shortening_Service_YP/internal/repository"
	"github.com/qso808/URL_Shortening_Service_YP/internal/service"
	"github.com/rs/zerolog"
)

func main() {
	// Загружаем конфигурацию из аргументов командной строки
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Создаем репозиторий
	repo := repository.NewMemoryRepository()

	// Создаем сервис
	shortenerService := service.NewShortenerService(repo)

	// Создаем хэндлер
	shortenerHandler := handler.NewShortenerHandler(shortenerService, cfg.BaseURL)

	// Инициализируем logger zerolog на уровне Info
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	// Настраиваем роутер с использованием chi
	router := chi.NewRouter()

	// Добавляем middleware для поддержки gzip (должен быть первым для обработки запросов/ответов)
	router.Use(customMiddleware.GzipMiddleware)

	// Добавляем кастомный middleware для логирования запросов и ответов
	router.Use(customMiddleware.RequestLogger(logger))
	router.Use(middleware.Recoverer)

	// POST / - сокращение URL (text/plain)
	router.Post("/", shortenerHandler.ShortenURL)

	// POST /api/shorten - сокращение URL (JSON)
	router.Post("/api/shorten", shortenerHandler.ShortenURLJSON)

	// GET /{id} - редирект на оригинальный URL
	router.Get("/{id}", shortenerHandler.Redirect)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on %s", cfg.ServerAddress)
		log.Printf("Base URL: %s", cfg.BaseURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
