package repository

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresRepository - PostgreSQL реализация репозитория
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgreSQL репозитория
// Выполняет миграции для создания необходимых таблиц
func NewPostgresRepository(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &PostgresRepository{
		db: db,
	}

	// Выполняем миграции
	if err := repo.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

// migrate выполняет миграции для создания таблиц
func (r *PostgresRepository) migrate() error {
	// Создаем таблицу для хранения сокращенных URL
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS url_mappings (
			id SERIAL PRIMARY KEY,
			short_url VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_short_url ON url_mappings(short_url);
	`

	if _, err := r.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// Save сохраняет связь между коротким ID и оригинальным URL
func (r *PostgresRepository) Save(id string, originalURL string) error {
	query := `
		INSERT INTO url_mappings (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url
	`

	_, err := r.db.Exec(query, id, originalURL)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}

	return nil
}

// SaveBatch сохраняет множество URL в одной транзакции
func (r *PostgresRepository) SaveBatch(mappings map[string]string) error {
	if len(mappings) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем statement
	stmt, err := tx.Prepare(`
		INSERT INTO url_mappings (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Выполняем вставки
	for shortID, originalURL := range mappings {
		if _, err := stmt.Exec(shortID, originalURL); err != nil {
			return fmt.Errorf("failed to save URL %s: %w", shortID, err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *PostgresRepository) Get(id string) (string, error) {
	query := `
		SELECT original_url
		FROM url_mappings
		WHERE short_url = $1
	`

	var originalURL string
	err := r.db.QueryRow(query, id).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("URL not found")
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	return originalURL, nil
}

// Close закрывает соединение с базой данных
func (r *PostgresRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

