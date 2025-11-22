package storage

import (
	"pr-review-assigner/internal/api"
)

// UserRepository предоставляет методы для работы с пользователями
type UserRepository struct {
	*Repository
}

// NewUserRepository создает новый экземпляр репозитория пользователей
func NewUserRepository(repo *Repository) *UserRepository {
	return &UserRepository{Repository: repo}
}

// CreateOrUpdateUser создает или обновляет пользователя
func (r *UserRepository) CreateOrUpdateUser(user *api.User) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, user.UserId, user.Username, user.TeamName, user.IsActive)
	if err != nil {
		return HandleDBError(err)
	}
	return nil
}

// GetUser получает пользователя по ID
func (r *UserRepository) GetUser(userID string) (*api.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`
	var user api.User
	err := r.db.QueryRow(query, userID).Scan(
		&user.UserId,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
	)
	if err != nil {
		return nil, HandleDBError(err)
	}
	return &user, nil
}

// UpdateUserIsActive обновляет флаг активности пользователя и возвращает обновленного пользователя
func (r *UserRepository) UpdateUserIsActive(userID string, isActive bool) (*api.User, error) {
	query := `
		UPDATE users
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2
		RETURNING user_id, username, team_name, is_active
	`
	var user api.User
	err := r.db.QueryRow(query, isActive, userID).Scan(
		&user.UserId,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
	)
	if err != nil {
		return nil, HandleDBError(err)
	}
	
	return &user, nil
}

// GetActiveUsersByTeam получает список активных пользователей команды, исключая указанного пользователя
func (r *UserRepository) GetActiveUsersByTeam(teamName string, excludeUserID string) ([]api.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY user_id
	`
	rows, err := r.db.Query(query, teamName, excludeUserID)
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var users []api.User
	for rows.Next() {
		var user api.User
		err := rows.Scan(
			&user.UserId,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
		)
		if err != nil {
			return nil, HandleDBError(err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return users, nil
}

