package service

import (
	"math/rand"
	"time"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

// PRService предоставляет бизнес-логику для работы с Pull Request'ами
type PRService struct {
	prRepo   storage.PRRepositoryInterface
	userRepo storage.UserRepositoryInterface
	teamRepo storage.TeamRepositoryInterface
}

// NewPRService создает новый экземпляр сервиса PR
func NewPRService(prRepo storage.PRRepositoryInterface, userRepo storage.UserRepositoryInterface, teamRepo storage.TeamRepositoryInterface) *PRService {
	return &PRService{
		prRepo:   prRepo,
		userRepo: userRepo,
		teamRepo: teamRepo,
	}
}

// CreatePR создает новый PR и автоматически назначает до 2 активных ревьюверов из команды автора
func (s *PRService) CreatePR(prID, prName, authorID string) (*api.PullRequest, error) {
	// Проверяем существование автора
	author, err := s.userRepo.GetUser(authorID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Проверяем, что PR еще не существует (попытка создать существующий PR)
	existingPR, err := s.prRepo.GetPR(prID)
	if err == nil && existingPR != nil {
		return nil, ErrPRExists
	}
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}

	// Получаем активных пользователей команды автора (исключая самого автора)
	candidates, err := s.userRepo.GetActiveUsersByTeam(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	// Выбираем случайных ревьюверов (до 2), исключая автора
	reviewerIDs := s.selectRandomReviewers(candidates, 2, authorID)

	// Создаем PR
	now := time.Now()
	pr := &api.PullRequest{
		PullRequestId:     prID,
		PullRequestName:   prName,
		AuthorId:          authorID,
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: reviewerIDs,
		CreatedAt:         &now,
	}

	err = s.prRepo.CreatePR(pr)
	if err != nil {
		if err == storage.ErrDuplicateKey {
			return nil, ErrPRExists
		}
		return nil, err
	}

	// Возвращаем созданный PR
	return s.prRepo.GetPR(prID)
}

// MergePR помечает PR как MERGED (идемпотентная операция)
func (s *PRService) MergePR(prID string) (*api.PullRequest, error) {
	// Получаем PR
	pr, err := s.prRepo.GetPR(prID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Если PR уже MERGED, возвращаем его
	if pr.Status == api.PullRequestStatusMERGED {
		return pr, nil
	}

	// Обновляем статус на MERGED
	now := time.Now()
	err = s.prRepo.UpdatePRStatus(prID, api.PullRequestStatusMERGED, &now)
	if err != nil {
		return nil, err
	}

	// Возвращаем обновленный PR
	return s.prRepo.GetPR(prID)
}

// ReassignReviewer переназначает одного ревьювера на другого из команды заменяемого ревьювера
// Работает только для OPEN PR
func (s *PRService) ReassignReviewer(prID, oldUserID string) (*api.PullRequest, string, error) {
	// Получаем PR
	pr, err := s.prRepo.GetPR(prID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	// Проверяем, что PR не MERGED
	if pr.Status == api.PullRequestStatusMERGED {
		return nil, "", ErrPRMerged
	}

	// Проверяем, что старый ревьювер назначен на этот PR
	isAssigned := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldUserID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return nil, "", ErrNotAssigned
	}

	// Получаем информацию о заменяемом ревьювере
	oldReviewer, err := s.userRepo.GetUser(oldUserID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	// Получаем активных пользователей команды заменяемого ревьювера (исключая его самого и уже назначенных ревьюверов)
	candidates, err := s.userRepo.GetActiveUsersByTeam(oldReviewer.TeamName, oldUserID)
	if err != nil {
		return nil, "", err
	}

	// Исключаем уже назначенных ревьюверов из кандидатов
	assignedMap := make(map[string]bool)
	for _, reviewerID := range pr.AssignedReviewers {
		assignedMap[reviewerID] = true
	}

	availableCandidates := make([]api.User, 0)
	for _, candidate := range candidates {
		if !assignedMap[candidate.UserId] {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	// Проверяем наличие доступных кандидатов
	if len(availableCandidates) == 0 {
		return nil, "", ErrNoCandidate
	}

	// Выбираем случайного нового ревьювера, исключая автора PR
	newReviewerIDs := s.selectRandomReviewers(availableCandidates, 1, pr.AuthorId)
	if len(newReviewerIDs) == 0 {
		return nil, "", ErrNoCandidate
	}
	newUserID := newReviewerIDs[0]

	// Переназначаем ревьювера
	err = s.prRepo.ReassignReviewer(prID, oldUserID, newUserID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, "", ErrNotAssigned
		}
		return nil, "", err
	}

	// Возвращаем обновленный PR и ID нового ревьювера
	updatedPR, err := s.prRepo.GetPR(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newUserID, nil
}

// GetPRsByReviewer получает список PR, где пользователь назначен ревьювером
func (s *PRService) GetPRsByReviewer(userID string) ([]api.PullRequestShort, error) {
	// Проверяем существование пользователя
	_, err := s.userRepo.GetUser(userID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Получаем список PR
	prs, err := s.prRepo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

// selectRandomReviewers выбирает случайных ревьюверов из списка кандидатов (до maxCount)
// Исключает из списка кандидатов автора PR
func (s *PRService) selectRandomReviewers(candidates []api.User, maxCount int, excludeUserID string) []string {
	// Фильтруем кандидатов, исключая автора PR
	filteredCandidates := make([]api.User, 0)
	for _, candidate := range candidates {
		if candidate.UserId != excludeUserID {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	if len(filteredCandidates) == 0 {
		return []string{}
	}

	// Если кандидатов меньше или равно maxCount, возвращаем всех
	if len(filteredCandidates) <= maxCount {
		result := make([]string, len(filteredCandidates))
		for i, user := range filteredCandidates {
			result[i] = user.UserId
		}
		return result
	}

	// Выбираем случайных ревьюверов
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	selected := make(map[int]bool)
	result := make([]string, 0, maxCount)

	for len(result) < maxCount {
		idx := rng.Intn(len(filteredCandidates))
		if !selected[idx] {
			selected[idx] = true
			result = append(result, filteredCandidates[idx].UserId)
		}
	}

	return result
}
