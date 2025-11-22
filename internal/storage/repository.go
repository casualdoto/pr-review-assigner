package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrDuplicateKey        = errors.New("duplicate key")
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

// HandleDBError обрабатывает ошибки БД и возвращает соответствующие ошибки приложения
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	// Проверка на ошибки PostgreSQL через тип pq.Error
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505": // unique_violation
			return ErrDuplicateKey
		case "23503": // foreign_key_violation
			return ErrForeignKeyViolation
		}
	}

	return fmt.Errorf("database error: %w", err)
}
