package service

import (
	"log"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// UserService предоставляет бизнес-логику для работы с пользователями
type UserService struct {
	userRepo storage.UserRepositoryInterface
	prRepo   storage.PRRepositoryInterface
	teamRepo storage.TeamRepositoryInterface
}

// NewUserService создает новый экземпляр сервиса пользователей
func NewUserService(userRepo storage.UserRepositoryInterface, prRepo storage.PRRepositoryInterface, teamRepo storage.TeamRepositoryInterface) *UserService {
	return &UserService{
		userRepo: userRepo,
		prRepo:   prRepo,
		teamRepo: teamRepo,
	}
}

// SetUserIsActive устанавливает флаг активности пользователя
// При деактивации автоматически переназначает все открытые PR, где пользователь является ревьювером
func (s *UserService) SetUserIsActive(userID string, isActive bool) (*api.User, error) {
	// Обновляем статус пользователя
	user, err := s.userRepo.UpdateUserIsActive(userID, isActive)
	if err != nil {
		return nil, MapStorageError(err)
	}

	// Если пользователь деактивирован, переназначаем его PR
	if !isActive {
		if err := s.reassignUserPRs(userID, user.TeamName); err != nil {
			log.Printf("Warning: failed to reassign PRs for user %s: %v", userID, err)
			// Не возвращаем ошибку, чтобы деактивация пользователя прошла успешно
		}
	}

	return user, nil
}

// reassignUserPRs переназначает все открытые PR, где пользователь является ревьювером
func (s *UserService) reassignUserPRs(userID string, teamName string) error {
	// Получаем все PR, где пользователь - ревьювер
	prs, err := s.prRepo.GetPRsByReviewer(userID)
	if err != nil {
		return err
	}

	// Фильтруем только OPEN PR
	for _, prShort := range prs {
		if prShort.Status != api.PullRequestShortStatusOPEN {
			continue
		}

		// Получаем полную информацию о PR
		pr, err := s.prRepo.GetPR(prShort.PullRequestId)
		if err != nil {
			log.Printf("Warning: failed to get PR %s: %v", prShort.PullRequestId, err)
			continue
		}

		// Получаем активных кандидатов из команды (исключая деактивированного пользователя)
		candidates, err := s.userRepo.GetActiveUsersByTeam(teamName, userID)
		if err != nil {
			log.Printf("Warning: failed to get candidates for PR %s: %v", prShort.PullRequestId, err)
			continue
		}

		// Исключаем автора и уже назначенных ревьюверов
		assignedMap := make(map[string]bool)
		for _, reviewerID := range pr.AssignedReviewers {
			if reviewerID != userID {
				assignedMap[reviewerID] = true
			}
		}
		assignedMap[pr.AuthorId] = true

		// Ищем доступного кандидата
		var newReviewerID string
		for _, candidate := range candidates {
			if !assignedMap[candidate.UserId] {
				newReviewerID = candidate.UserId
				break
			}
		}

		// Если нет доступных кандидатов, просто удаляем ревьювера
		if newReviewerID == "" {
			log.Printf("Warning: no available candidates for PR %s, removing reviewer %s", prShort.PullRequestId, userID)
			_, err = s.prRepo.ReassignReviewer(prShort.PullRequestId, userID, "")
			if err != nil {
				log.Printf("Warning: failed to remove reviewer from PR %s: %v", prShort.PullRequestId, err)
			}
			continue
		}

		// Переназначаем ревьювера
		_, err = s.prRepo.ReassignReviewer(prShort.PullRequestId, userID, newReviewerID)
		if err != nil {
			log.Printf("Warning: failed to reassign PR %s: %v", prShort.PullRequestId, err)
			continue
		}

		log.Printf("Successfully reassigned PR %s: %s -> %s", prShort.PullRequestId, userID, newReviewerID)
	}

	return nil
}
