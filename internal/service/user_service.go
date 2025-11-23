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

// DeactivateTeamUsers массово деактивирует пользователей команды и переназначает их открытые PR
func (s *UserService) DeactivateTeamUsers(teamName string, userIDs []string) ([]api.User, int, error) {
	if len(userIDs) == 0 {
		return []api.User{}, 0, nil
	}

	// Проверяем существование команды
	_, err := s.teamRepo.GetTeam(teamName)
	if err != nil {
		return nil, 0, MapStorageError(err)
	}

	// Получаем всех пользователей команды для валидации
	teamUsers, err := s.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, 0, MapStorageError(err)
	}

	// Создаем карту пользователей команды
	teamUserMap := make(map[string]bool)
	for _, user := range teamUsers {
		teamUserMap[user.UserId] = true
	}

	// Валидируем, что все userIDs принадлежат команде
	for _, userID := range userIDs {
		if !teamUserMap[userID] {
			return nil, 0, ErrNotFound
		}
	}

	// Получаем всех активных пользователей команды (для кандидатов на переназначение)
	// Исключаем тех, кого собираемся деактивировать
	deactivatingMap := make(map[string]bool)
	for _, userID := range userIDs {
		deactivatingMap[userID] = true
	}

	var activeCandidates []api.User
	for _, user := range teamUsers {
		if user.IsActive && !deactivatingMap[user.UserId] {
			activeCandidates = append(activeCandidates, user)
		}
	}

	// Получаем все открытые PR деактивируемых пользователей одним запросом
	openPRs, err := s.prRepo.GetOpenPRsByReviewers(userIDs)
	if err != nil {
		return nil, 0, MapStorageError(err)
	}

	// Подготавливаем план переназначений в памяти
	reassignments := make(map[string]map[string]string) // prID -> {oldUserID -> newUserID}
	reassignedCount := 0

	for _, pr := range openPRs {
		// Создаем карту уже назначенных ревьюверов на этот PR
		assignedMap := make(map[string]bool)
		assignedMap[pr.AuthorId] = true // Автор не может быть ревьювером
		for _, reviewerID := range pr.AssignedReviewers {
			if !deactivatingMap[reviewerID] {
				// Ревьювер остается, добавляем в карту
				assignedMap[reviewerID] = true
			}
		}

		// Для каждого деактивируемого ревьювера в этом PR
		for _, reviewerID := range pr.AssignedReviewers {
			if deactivatingMap[reviewerID] {
				// Ищем кандидата на замену
				var newReviewerID string
				for _, candidate := range activeCandidates {
					if !assignedMap[candidate.UserId] {
						newReviewerID = candidate.UserId
						assignedMap[candidate.UserId] = true // Помечаем как назначенного
						break
					}
				}

				// Добавляем в план переназначений (даже если newReviewerID пустой - тогда просто удалим)
				if reassignments[pr.PullRequestId] == nil {
					reassignments[pr.PullRequestId] = make(map[string]string)
				}
				reassignments[pr.PullRequestId][reviewerID] = newReviewerID
				reassignedCount++

				if newReviewerID == "" {
					log.Printf("Warning: no available candidates for PR %s, will remove reviewer %s", pr.PullRequestId, reviewerID)
				}
			}
		}
	}

	// Выполняем массовую деактивацию и переназначение
	deactivatedUsers, err := s.userRepo.BatchDeactivateUsers(userIDs)
	if err != nil {
		return nil, 0, MapStorageError(err)
	}

	// Выполняем массовое переназначение PR
	if len(reassignments) > 0 {
		err = s.prRepo.BatchReassignReviewers(reassignments)
		if err != nil {
			log.Printf("Warning: failed to batch reassign PRs: %v", err)
			// Не возвращаем ошибку, так как пользователи уже деактивированы
		}
	}

	log.Printf("Successfully deactivated %d users and reassigned %d PR assignments", len(deactivatedUsers), reassignedCount)

	return deactivatedUsers, reassignedCount, nil
}
