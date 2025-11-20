package storage

import (
	"database/sql"
	"time"

	"pr-review-assigner/internal/api"
)

// PRRepository предоставляет методы для работы с Pull Requests
type PRRepository struct {
	*Repository
}

// NewPRRepository создает новый экземпляр репозитория PR
func NewPRRepository(repo *Repository) *PRRepository {
	return &PRRepository{Repository: repo}
}

// CreatePR создает новый Pull Request
func (r *PRRepository) CreatePR(pr *api.PullRequest) error {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	createdAt := time.Now()
	if pr.CreatedAt != nil {
		createdAt = *pr.CreatedAt
	}
	
	_, err := r.db.Exec(query, pr.PullRequestId, pr.PullRequestName, pr.AuthorId, string(pr.Status), createdAt)
	if err != nil {
		return HandleDBError(err)
	}
	
	// Назначаем ревьюверов, если они указаны
	if len(pr.AssignedReviewers) > 0 {
		err = r.AssignReviewers(pr.PullRequestId, pr.AssignedReviewers)
		if err != nil {
			return HandleDBError(err)
		}
	}
	
	return nil
}

// GetPR получает Pull Request по ID со всеми назначенными ревьюверами
func (r *PRRepository) GetPR(prID string) (*api.PullRequest, error) {
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	var pr api.PullRequest
	var createdAt, mergedAt sql.NullTime
	
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestId,
		&pr.PullRequestName,
		&pr.AuthorId,
		&pr.Status,
		&createdAt,
		&mergedAt,
	)
	if err != nil {
		return nil, HandleDBError(err)
	}
	
	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}
	
	// Получаем назначенных ревьюверов
	reviewers, err := r.GetReviewersByPR(prID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	pr.AssignedReviewers = reviewers
	
	return &pr, nil
}

// UpdatePRStatus обновляет статус PR и устанавливает merged_at при необходимости
func (r *PRRepository) UpdatePRStatus(prID string, status api.PullRequestStatus, mergedAt *time.Time) error {
	var query string
	var err error
	
	if status == api.PullRequestStatusMERGED && mergedAt != nil {
		query = `
			UPDATE pull_requests
			SET status = $1, merged_at = $2
			WHERE pull_request_id = $3
		`
		_, err = r.db.Exec(query, string(status), mergedAt, prID)
	} else {
		query = `
			UPDATE pull_requests
			SET status = $1
			WHERE pull_request_id = $2
		`
		_, err = r.db.Exec(query, string(status), prID)
	}
	
	if err != nil {
		return HandleDBError(err)
	}
	
	return nil
}

// GetPRsByReviewer получает список PR, где пользователь назначен ревьювером
func (r *PRRepository) GetPRsByReviewer(userID string) ([]api.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var prs []api.PullRequestShort
	for rows.Next() {
		var pr api.PullRequestShort
		err := rows.Scan(
			&pr.PullRequestId,
			&pr.PullRequestName,
			&pr.AuthorId,
			&pr.Status,
		)
		if err != nil {
			return nil, HandleDBError(err)
		}
		prs = append(prs, pr)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return prs, nil
}

// AssignReviewers назначает ревьюверов на PR
func (r *PRRepository) AssignReviewers(prID string, reviewerIDs []string) error {
	if len(reviewerIDs) == 0 {
		return nil
	}
	
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (pull_request_id, user_id) DO NOTHING
	`
	
	for _, reviewerID := range reviewerIDs {
		_, err := r.db.Exec(query, prID, reviewerID)
		if err != nil {
			return HandleDBError(err)
		}
	}
	
	return nil
}

// ReassignReviewer переназначает одного ревьювера на другого
func (r *PRRepository) ReassignReviewer(prID string, oldUserID, newUserID string) error {
	// Проверяем, что старый ревьювер назначен на этот PR
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`
	err := r.db.QueryRow(checkQuery, prID, oldUserID).Scan(&exists)
	if err != nil {
		return HandleDBError(err)
	}
	if !exists {
		return ErrNotFound
	}
	
	// Удаляем старого ревьювера и добавляем нового в одной транзакции
	tx, err := r.db.Begin()
	if err != nil {
		return HandleDBError(err)
	}
	defer tx.Rollback()
	
	// Удаляем старого ревьювера
	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	_, err = tx.Exec(deleteQuery, prID, oldUserID)
	if err != nil {
		return HandleDBError(err)
	}
	
	// Добавляем нового ревьювера
	insertQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
	_, err = tx.Exec(insertQuery, prID, newUserID)
	if err != nil {
		return HandleDBError(err)
	}
	
	if err = tx.Commit(); err != nil {
		return HandleDBError(err)
	}
	
	return nil
}

// GetReviewersByPR получает список ревьюверов для PR
func (r *PRRepository) GetReviewersByPR(prID string) ([]string, error) {
	query := `
		SELECT user_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`
	rows, err := r.db.Query(query, prID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		err := rows.Scan(&reviewerID)
		if err != nil {
			return nil, HandleDBError(err)
		}
		reviewers = append(reviewers, reviewerID)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return reviewers, nil
}

