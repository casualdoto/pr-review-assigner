package storage

import (
	"pr-review-assigner/internal/api"
)

// TeamRepository предоставляет методы для работы с командами
type TeamRepository struct {
	*Repository
}

// NewTeamRepository создает новый экземпляр репозитория команд
func NewTeamRepository(repo *Repository) *TeamRepository {
	return &TeamRepository{Repository: repo}
}

// CreateTeam создает новую команду
func (r *TeamRepository) CreateTeam(teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err := r.db.Exec(query, teamName)
	if err != nil {
		return HandleDBError(err)
	}
	return nil
}

// GetTeam получает команду с участниками по имени
func (r *TeamRepository) GetTeam(teamName string) (*api.Team, error) {
	// Сначала проверяем существование команды
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	err := r.db.QueryRow(checkQuery, teamName).Scan(&exists)
	if err != nil {
		return nil, HandleDBError(err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	// Получаем участников команды
	query := `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, HandleDBError(err)
	}
	defer rows.Close()

	var members []api.TeamMember
	for rows.Next() {
		var member api.TeamMember
		err := rows.Scan(&member.UserId, &member.Username, &member.IsActive)
		if err != nil {
			return nil, HandleDBError(err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, HandleDBError(err)
	}

	return &api.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

// TeamExists проверяет существование команды
func (r *TeamRepository) TeamExists(teamName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	err := r.db.QueryRow(query, teamName).Scan(&exists)
	if err != nil {
		return false, HandleDBError(err)
	}
	return exists, nil
}

