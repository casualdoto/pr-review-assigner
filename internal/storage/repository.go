package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDuplicateKey  = errors.New("duplicate key")
	ErrForeignKeyViolation = errors.New("foreign key violation")
)

// Repository представляет базовый репозиторий для работы с БД
type Repository struct {
	db *sql.DB
}

// NewRepository создает новый экземпляр репозитория
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB возвращает подключение к БД
func (r *Repository) DB() *sql.DB {
	return r.db
}

// BeginTx начинает транзакцию
func (r *Repository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// HandleDBError обрабатывает ошибки БД и возвращает соответствующие ошибки приложения
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}
	
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	
	// Проверка на duplicate key error
	if err.Error() == "pq: duplicate key value violates unique constraint" {
		return ErrDuplicateKey
	}
	
	// Проверка на foreign key violation
	if err.Error() == "pq: insert or update on table violates foreign key constraint" {
		return ErrForeignKeyViolation
	}
	
	return fmt.Errorf("database error: %w", err)
}

