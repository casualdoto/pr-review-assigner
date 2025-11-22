package service

import (
	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// UserService предоставляет бизнес-логику для работы с пользователями
type UserService struct {
	userRepo storage.UserRepositoryInterface
}

// NewUserService создает новый экземпляр сервиса пользователей
func NewUserService(userRepo storage.UserRepositoryInterface) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// SetUserIsActive устанавливает флаг активности пользователя
func (s *UserService) SetUserIsActive(userID string, isActive bool) (*api.User, error) {
	user, err := s.userRepo.UpdateUserIsActive(userID, isActive)
	if err != nil {
		return nil, MapStorageError(err)
	}

	return user, nil
}
