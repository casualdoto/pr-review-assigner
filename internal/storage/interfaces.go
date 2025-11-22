package storage

import (
	"time"

	"pr-review-assigner/internal/api"
)

// TeamRepositoryInterface определяет интерфейс для работы с командами
type TeamRepositoryInterface interface {
	CreateTeam(teamName string) error
	GetTeam(teamName string) (*api.Team, error)
	TeamExists(teamName string) (bool, error)
}

// UserRepositoryInterface определяет интерфейс для работы с пользователями
type UserRepositoryInterface interface {
	CreateOrUpdateUser(user *api.User) error
	GetUser(userID string) (*api.User, error)
	UpdateUserIsActive(userID string, isActive bool) (*api.User, error)
	GetActiveUsersByTeam(teamName string, excludeUserID string) ([]api.User, error)
}

// PRRepositoryInterface определяет интерфейс для работы с Pull Requests
type PRRepositoryInterface interface {
	CreatePR(pr *api.PullRequest) (*api.PullRequest, error)
	GetPR(prID string) (*api.PullRequest, error)
	UpdatePRStatus(prID string, status api.PullRequestStatus, mergedAt *time.Time) (*api.PullRequest, error)
	GetPRsByReviewer(userID string) ([]api.PullRequestShort, error)
	ReassignReviewer(prID string, oldUserID, newUserID string) (*api.PullRequest, error)
}
