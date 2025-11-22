package service

import (
	"errors"
	"math/rand"
	"time"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"
)

const (
	// MaxReviewers максимальное количество ревьюверов на PR
	MaxReviewers = 2
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

// CreatePR создает новый PR и автоматически назначает до MaxReviewers активных ревьюверов из команды автора
func (s *PRService) CreatePR(prID, prName, authorID string) (*api.PullRequest, error) {
	// Проверяем существование автора
	author, err := s.userRepo.GetUser(authorID)
	if err != nil {
		return nil, MapStorageError(err)
	}

	// Проверяем, что PR еще не существует (попытка создать существующий PR)
	existingPR, err := s.prRepo.GetPR(prID)
	if err == nil && existingPR != nil {
		return nil, ErrPRExists
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, MapStorageError(err)
	}

	// Получаем активных пользователей команды автора (исключая самого автора)
	candidates, err := s.userRepo.GetActiveUsersByTeam(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	// Выбираем случайных ревьюверов (до MaxReviewers)
	reviewerIDs := s.selectRandomReviewers(candidates, MaxReviewers)

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

	createdPR, err := s.prRepo.CreatePR(pr)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateKey) {
			return nil, ErrPRExists
		}
		return nil, MapStorageError(err)
	}

	return createdPR, nil
}

// MergePR помечает PR как MERGED (идемпотентная операция)
func (s *PRService) MergePR(prID string) (*api.PullRequest, error) {
	// Получаем PR
	pr, err := s.prRepo.GetPR(prID)
	if err != nil {
		return nil, MapStorageError(err)
	}

	// Если PR уже MERGED, возвращаем его
	if pr.Status == api.PullRequestStatusMERGED {
		return pr, nil
	}

	// Обновляем статус на MERGED
	now := time.Now()
	updatedPR, err := s.prRepo.UpdatePRStatus(prID, api.PullRequestStatusMERGED, &now)
	if err != nil {
		return nil, MapStorageError(err)
	}

	return updatedPR, nil
}

// ReassignReviewer переназначает одного ревьювера на другого из команды заменяемого ревьювера
// Работает только для OPEN PR
func (s *PRService) ReassignReviewer(prID, oldUserID string) (*api.PullRequest, string, error) {
	// Получаем PR
	pr, err := s.prRepo.GetPR(prID)
	if err != nil {
		return nil, "", MapStorageError(err)
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
		return nil, "", MapStorageError(err)
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

	// Выбираем случайного нового ревьювера
	newReviewerIDs := s.selectRandomReviewers(availableCandidates, 1)
	if len(newReviewerIDs) == 0 {
		return nil, "", ErrNoCandidate
	}
	newUserID := newReviewerIDs[0]

	// Переназначаем ревьювера
	updatedPR, err := s.prRepo.ReassignReviewer(prID, oldUserID, newUserID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, "", ErrNotAssigned
		}
		return nil, "", MapStorageError(err)
	}

	return updatedPR, newUserID, nil
}

// GetPRsByReviewer получает список PR, где пользователь назначен ревьювером
func (s *PRService) GetPRsByReviewer(userID string) ([]api.PullRequestShort, error) {
	// Проверяем существование пользователя
	_, err := s.userRepo.GetUser(userID)
	if err != nil {
		return nil, MapStorageError(err)
	}

	// Получаем список PR
	prs, err := s.prRepo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

// selectRandomReviewers выбирает случайных ревьюверов из списка кандидатов (до maxCount)
func (s *PRService) selectRandomReviewers(candidates []api.User, maxCount int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	// Если кандидатов меньше или равно maxCount, возвращаем всех
	if len(candidates) <= maxCount {
		result := make([]string, len(candidates))
		for i, user := range candidates {
			result[i] = user.UserId
		}
		return result
	}

	// Выбираем случайных ревьюверов
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	selected := make(map[int]bool)
	result := make([]string, 0, maxCount)

	for len(result) < maxCount {
		idx := rng.Intn(len(candidates))
		if !selected[idx] {
			selected[idx] = true
			result = append(result, candidates[idx].UserId)
		}
	}

	return result
}
