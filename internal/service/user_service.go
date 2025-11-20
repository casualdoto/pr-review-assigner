package service

import (
	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// UserService предоставляет бизнес-логику для работы с пользователями
type UserService struct {
	userRepo *storage.UserRepository
}

// NewUserService создает новый экземпляр сервиса пользователей
func NewUserService(userRepo *storage.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// SetUserIsActive устанавливает флаг активности пользователя
func (s *UserService) SetUserIsActive(userID string, isActive bool) (*api.User, error) {
	err := s.userRepo.UpdateUserIsActive(userID, isActive)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Возвращаем обновленного пользователя
	user, err := s.userRepo.GetUser(userID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return user, nil
}

