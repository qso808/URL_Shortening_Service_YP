package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
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
			user_id VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_short_url ON url_mappings(short_url);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url ON url_mappings(original_url);
		CREATE INDEX IF NOT EXISTS idx_user_id ON url_mappings(user_id);
	`
	if _, err := r.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	// Миграция: добавляем колонку user_id, если её ещё нет (для существующих БД)
	alterSQL := `ALTER TABLE url_mappings ADD COLUMN IF NOT EXISTS user_id VARCHAR(255) NOT NULL DEFAULT '';`
	if _, err := r.db.Exec(alterSQL); err != nil {
		return fmt.Errorf("failed to add user_id column: %w", err)
	}
	createIdxUser := `CREATE INDEX IF NOT EXISTS idx_user_id ON url_mappings(user_id);`
	if _, err := r.db.Exec(createIdxUser); err != nil {
		return fmt.Errorf("failed to create idx_user_id: %w", err)
	}
	// Миграция: добавляем колонку is_deleted для soft delete
	alterDeleted := `ALTER TABLE url_mappings ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;`
	if _, err := r.db.Exec(alterDeleted); err != nil {
		return fmt.Errorf("failed to add is_deleted column: %w", err)
	}
	return nil
}

// ErrDuplicateURL - ошибка, возникающая при попытке сохранить уже существующий URL
var ErrDuplicateURL = errors.New("duplicate URL")

// Save сохраняет связь между коротким ID, оригинальным URL и user_id
func (r *PostgresRepository) Save(id string, originalURL string, userID string) error {
	query := `
		INSERT INTO url_mappings (short_url, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url, user_id = EXCLUDED.user_id
	`

	_, err := r.db.Exec(query, id, originalURL, userID)
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

// SaveBatch сохраняет множество URL в одной транзакции (userID для всех записей)
func (r *PostgresRepository) SaveBatch(mappings map[string]string, userID string) error {
	if len(mappings) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO url_mappings (short_url, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url, user_id = EXCLUDED.user_id
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for shortID, originalURL := range mappings {
		if _, err := stmt.Exec(shortID, originalURL, userID); err != nil {
			return fmt.Errorf("failed to save URL %s: %w", shortID, err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Get возвращает оригинальный URL по короткому ID. ErrNotFound / ErrDeleted — для выбора status code в хендлере.
func (r *PostgresRepository) Get(id string) (string, error) {
	query := `
		SELECT original_url, COALESCE(is_deleted, FALSE)
		FROM url_mappings
		WHERE short_url = $1
	`

	var originalURL string
	var isDeleted bool
	err := r.db.QueryRow(query, id).Scan(&originalURL, &isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	if isDeleted {
		return "", ErrDeleted
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

// GetByUserID возвращает все не удалённые URL, сокращённые пользователем userID
func (r *PostgresRepository) GetByUserID(userID string) ([]UserURL, error) {
	query := `
		SELECT short_url, original_url
		FROM url_mappings
		WHERE user_id = $1 AND (is_deleted IS NULL OR is_deleted = FALSE)
		ORDER BY created_at
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs by user: %w", err)
	}
	defer rows.Close()
	var result []UserURL
	for rows.Next() {
		var shortURL, originalURL string
		if err := rows.Scan(&shortURL, &originalURL); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, UserURL{ShortURL: shortURL, OriginalURL: originalURL})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}

// MarkDeleted помечает указанные short_url как удалённые только для записей, принадлежащих userID (batch update)
func (r *PostgresRepository) MarkDeleted(userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	query := `
		UPDATE url_mappings
		SET is_deleted = TRUE
		WHERE user_id = $1 AND short_url = ANY($2)
	`
	_, err := r.db.Exec(query, userID, pq.Array(shortIDs))
	if err != nil {
		return fmt.Errorf("failed to mark URLs as deleted: %w", err)
	}
	return nil
}

// Close закрывает соединение с базой данных
func (r *PostgresRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
