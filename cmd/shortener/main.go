package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qso808/URL_Shortening_Service_YP/internal/handler"
	"github.com/qso808/URL_Shortening_Service_YP/internal/repository"
	"github.com/qso808/URL_Shortening_Service_YP/internal/service"
)

const (
	defaultServerAddr = "localhost:8080"
	defaultBaseURL    = "http://localhost:8080"
)

func main() {
	// Создаем репозиторий
	repo := repository.NewMemoryRepository()

	// Создаем сервис
	shortenerService := service.NewShortenerService(repo)

	// Создаем хэндлер
	shortenerHandler := handler.NewShortenerHandler(shortenerService, defaultBaseURL)

	// Настраиваем роутер
	mux := http.NewServeMux()

	// POST / - сокращение URL
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			shortenerHandler.ShortenURL(w, r)
		} else {
			shortenerHandler.Redirect(w, r)
		}
	})

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    defaultServerAddr,
		Handler: mux,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on %s", defaultServerAddr)
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
