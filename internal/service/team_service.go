package service

import (
	"errors"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// TeamService предоставляет бизнес-логику для работы с командами
type TeamService struct {
	teamRepo storage.TeamRepositoryInterface
	userRepo storage.UserRepositoryInterface
}

// NewTeamService создает новый экземпляр сервиса команд
func NewTeamService(teamRepo storage.TeamRepositoryInterface, userRepo storage.UserRepositoryInterface) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// CreateOrUpdateTeam создает или обновляет команду с участниками
// Если команда уже существует, возвращает ErrTeamExists
// Создает/обновляет всех пользователей из списка участников
func (s *TeamService) CreateOrUpdateTeam(team *api.Team) (*api.Team, error) {
	// Создаем команду
	err := s.teamRepo.CreateTeam(team.TeamName)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateKey) {
			return nil, ErrTeamExists
		}
		return nil, MapStorageError(err)
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
		return nil, MapStorageError(err)
	}
	return team, nil
}
