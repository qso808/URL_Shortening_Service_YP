package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
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
		CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url ON url_mappings(original_url);
	`

	if _, err := r.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// ErrDuplicateURL - ошибка, возникающая при попытке сохранить уже существующий URL
var ErrDuplicateURL = errors.New("duplicate URL")

// Save сохраняет связь между коротким ID и оригинальным URL
func (r *PostgresRepository) Save(id string, originalURL string) error {
	query := `
		INSERT INTO url_mappings (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url
	`

	_, err := r.db.Exec(query, id, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка нарушением уникального ограничения на original_url
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == pgerrcode.UniqueViolation {
				// Проверяем, какое именно ограничение было нарушено
				// Может быть idx_original_url (наш индекс) или автоматически сгенерированное имя
				if pqErr.Constraint == "idx_original_url" || 
				   pqErr.Constraint == "url_mappings_original_url_key" ||
				   (pqErr.Column == "original_url" && pqErr.Table == "url_mappings") {
					return ErrDuplicateURL
				}
			}
		}
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

// GetByOriginalURL возвращает короткий ID по оригинальному URL
func (r *PostgresRepository) GetByOriginalURL(originalURL string) (string, error) {
	query := `
		SELECT short_url
		FROM url_mappings
		WHERE original_url = $1
	`

	var shortURL string
	err := r.db.QueryRow(query, originalURL).Scan(&shortURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("URL not found")
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	return shortURL, nil
}

// Close закрывает соединение с базой данных
func (r *PostgresRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

