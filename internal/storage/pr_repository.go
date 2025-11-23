package storage

import (
	"database/sql"
	"time"

	"pr-review-assigner/internal/api"

	"github.com/lib/pq"
)

// PRRepository предоставляет методы для работы с Pull Requests
type PRRepository struct {
	*Repository
}

// NewPRRepository создает новый экземпляр репозитория PR
func NewPRRepository(repo *Repository) *PRRepository {
	return &PRRepository{Repository: repo}
}

// CreatePR создает новый Pull Request и возвращает созданный PR
func (r *PRRepository) CreatePR(pr *api.PullRequest) (*api.PullRequest, error) {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`
	createdAt := time.Now()
	if pr.CreatedAt != nil {
		createdAt = *pr.CreatedAt
	}

	var createdPR api.PullRequest
	var createdAtTime, mergedAtTime sql.NullTime

	err := r.db.QueryRow(query, pr.PullRequestId, pr.PullRequestName, pr.AuthorId, string(pr.Status), createdAt).Scan(
		&createdPR.PullRequestId,
		&createdPR.PullRequestName,
		&createdPR.AuthorId,
		&createdPR.Status,
		&createdAtTime,
		&mergedAtTime,
	)
	if err != nil {
		return nil, HandleDBError(err)
	}

	if createdAtTime.Valid {
		createdPR.CreatedAt = &createdAtTime.Time
	}
	if mergedAtTime.Valid {
		createdPR.MergedAt = &mergedAtTime.Time
	}

	// Назначаем ревьюверов, если они указаны
	if len(pr.AssignedReviewers) > 0 {
		err = r.assignReviewers(pr.PullRequestId, pr.AssignedReviewers)
		if err != nil {
			return nil, HandleDBError(err)
		}
		createdPR.AssignedReviewers = pr.AssignedReviewers
	} else {
		createdPR.AssignedReviewers = []string{}
	}

	return &createdPR, nil
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
	reviewers, err := r.getReviewersByPR(prID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

// UpdatePRStatus обновляет статус PR и возвращает обновленный PR
func (r *PRRepository) UpdatePRStatus(prID string, status api.PullRequestStatus, mergedAt *time.Time) (*api.PullRequest, error) {
	var query string
	var pr api.PullRequest
	var createdAtTime, mergedAtTime sql.NullTime
	var err error

	if status == api.PullRequestStatusMERGED && mergedAt != nil {
		query = `
			UPDATE pull_requests
			SET status = $1, merged_at = $2
			WHERE pull_request_id = $3
			RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		`
		err = r.db.QueryRow(query, string(status), mergedAt, prID).Scan(
			&pr.PullRequestId,
			&pr.PullRequestName,
			&pr.AuthorId,
			&pr.Status,
			&createdAtTime,
			&mergedAtTime,
		)
	} else {
		query = `
			UPDATE pull_requests
			SET status = $1
			WHERE pull_request_id = $2
			RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		`
		err = r.db.QueryRow(query, string(status), prID).Scan(
			&pr.PullRequestId,
			&pr.PullRequestName,
			&pr.AuthorId,
			&pr.Status,
			&createdAtTime,
			&mergedAtTime,
		)
	}

	if err != nil {
		return nil, HandleDBError(err)
	}

	if createdAtTime.Valid {
		pr.CreatedAt = &createdAtTime.Time
	}
	if mergedAtTime.Valid {
		pr.MergedAt = &mergedAtTime.Time
	}

	// Получаем назначенных ревьюверов
	reviewers, err := r.getReviewersByPR(prID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
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

// assignReviewers назначает ревьюверов на PR
func (r *PRRepository) assignReviewers(prID string, reviewerIDs []string) error {
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

// ReassignReviewer переназначает одного ревьювера на другого и возвращает обновленный PR
// Если newUserID пустой, то просто удаляет старого ревьювера без назначения нового
func (r *PRRepository) ReassignReviewer(prID string, oldUserID, newUserID string) (*api.PullRequest, error) {
	// Проверяем, что старый ревьювер назначен на этот PR
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`
	err := r.db.QueryRow(checkQuery, prID, oldUserID).Scan(&exists)
	if err != nil {
		return nil, HandleDBError(err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	// Удаляем старого ревьювера и добавляем нового в одной транзакции
	tx, err := r.db.Begin()
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer tx.Rollback()

	// Удаляем старого ревьювера
	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	_, err = tx.Exec(deleteQuery, prID, oldUserID)
	if err != nil {
		return nil, HandleDBError(err)
	}

	// Добавляем нового ревьювера, если он указан
	if newUserID != "" {
		insertQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
		_, err = tx.Exec(insertQuery, prID, newUserID)
		if err != nil {
			return nil, HandleDBError(err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, HandleDBError(err)
	}

	// Возвращаем обновленный PR
	return r.GetPR(prID)
}

// AddReviewer добавляет ревьювера к PR
func (r *PRRepository) AddReviewer(prID string, userID string) error {
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (pull_request_id, user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, prID, userID)
	if err != nil {
		return HandleDBError(err)
	}
	return nil
}

// getReviewersByPR получает список ревьюверов для PR
func (r *PRRepository) getReviewersByPR(prID string) ([]string, error) {
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

// GetReviewerStatistics получает статистику по назначениям ревьюверов
func (r *PRRepository) GetReviewerStatistics() ([]ReviewerStatistic, error) {
	query := `
		SELECT u.user_id, u.username, COUNT(pr.pull_request_id) as assignments_count
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.user_id = pr.user_id
		GROUP BY u.user_id, u.username
		ORDER BY assignments_count DESC, u.username
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var statistics []ReviewerStatistic
	for rows.Next() {
		var stat ReviewerStatistic
		err := rows.Scan(&stat.UserID, &stat.Username, &stat.AssignmentsCount)
		if err != nil {
			return nil, HandleDBError(err)
		}
		statistics = append(statistics, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return statistics, nil
}

// GetOpenPRsByReviewers получает все открытые PR, где указанные пользователи являются ревьюверами
func (r *PRRepository) GetOpenPRsByReviewers(userIDs []string) ([]api.PullRequest, error) {
	if len(userIDs) == 0 {
		return []api.PullRequest{}, nil
	}

	query := `
		SELECT DISTINCT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = ANY($1) AND pr.status = 'OPEN'
		ORDER BY pr.created_at
	`
	rows, err := r.db.Query(query, pq.Array(userIDs))
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var prs []api.PullRequest
	for rows.Next() {
		var pr api.PullRequest
		var createdAt, mergedAt sql.NullTime

		err := rows.Scan(
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

		// Получаем ревьюверов для каждого PR
		reviewers, err := r.getReviewersByPR(pr.PullRequestId)
		if err != nil {
			return nil, HandleDBError(err)
		}
		pr.AssignedReviewers = reviewers

		prs = append(prs, pr)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return prs, nil
}

// BatchReassignReviewers массово переназначает ревьюверов в одной транзакции
// reassignments - карта: prID -> {oldUserID -> newUserID}
// Если newUserID пустой, ревьювер просто удаляется
func (r *PRRepository) BatchReassignReviewers(reassignments map[string]map[string]string) error {
	if len(reassignments) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return HandleDBError(err)
	}
	defer tx.Rollback()

	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	insertQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	for prID, changes := range reassignments {
		for oldUserID, newUserID := range changes {
			// Удаляем старого ревьювера
			_, err = tx.Exec(deleteQuery, prID, oldUserID)
			if err != nil {
				return HandleDBError(err)
			}

			// Добавляем нового ревьювера, если он указан
			if newUserID != "" {
				_, err = tx.Exec(insertQuery, prID, newUserID)
				if err != nil {
					return HandleDBError(err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return HandleDBError(err)
	}

	return nil
}
