package handler

import (
	"database/sql"
	"net/http"
)

// PingHandler обрабатывает GET запрос /ping для проверки соединения с базой данных
type PingHandler struct {
	db *sql.DB
}

// NewPingHandler создает новый экземпляр хендлера для проверки соединения с БД
func NewPingHandler(db *sql.DB) *PingHandler {
	return &PingHandler{
		db: db,
	}
}

// Ping проверяет соединение с базой данных
func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Если база данных не настроена, возвращаем 500
	if h.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Проверяем соединение с базой данных
	if err := h.db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Соединение успешно
	w.WriteHeader(http.StatusOK)
}

