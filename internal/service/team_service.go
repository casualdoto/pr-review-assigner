package service

import (
	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// TeamService предоставляет бизнес-логику для работы с командами
type TeamService struct {
	teamRepo *storage.TeamRepository
	userRepo *storage.UserRepository
}

// NewTeamService создает новый экземпляр сервиса команд
func NewTeamService(teamRepo *storage.TeamRepository, userRepo *storage.UserRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// CreateOrUpdateTeam создает или обновляет команду с участниками
// Если команда уже существует, возвращает ErrTeamExists
// Создает/обновляет всех пользователей из списка участников
func (s *TeamService) CreateOrUpdateTeam(team *api.Team) (*api.Team, error) {
	// Проверяем существование команды
	exists, err := s.teamRepo.TeamExists(team.TeamName)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrTeamExists
	}

	// Создаем команду
	err = s.teamRepo.CreateTeam(team.TeamName)
	if err != nil {
		if err == storage.ErrDuplicateKey {
			return nil, ErrTeamExists
		}
		return nil, err
	}

	// Создаем/обновляем всех участников команды
	for _, member := range team.Members {
		user := &api.User{
			UserId:   member.UserId,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		err = s.userRepo.CreateOrUpdateUser(user)
		if err != nil {
			return nil, err
		}
	}

	// Возвращаем созданную команду
	return s.GetTeam(team.TeamName)
}

// GetTeam получает команду с участниками
func (s *TeamService) GetTeam(teamName string) (*api.Team, error) {
	team, err := s.teamRepo.GetTeam(teamName)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return team, nil
}

