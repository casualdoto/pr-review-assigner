package service

import (
	"time"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/mock"
)

// MockTeamRepository - мок для TeamRepository
type MockTeamRepository struct {
	mock.Mock
}

func (m *MockTeamRepository) CreateTeam(teamName string) error {
	args := m.Called(teamName)
	return args.Error(0)
}

func (m *MockTeamRepository) GetTeam(teamName string) (*api.Team, error) {
	args := m.Called(teamName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.Team), args.Error(1)
}

func (m *MockTeamRepository) TeamExists(teamName string) (bool, error) {
	args := m.Called(teamName)
	return args.Bool(0), args.Error(1)
}

// MockUserRepository - мок для UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateOrUpdateUser(user *api.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUser(userID string) (*api.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUserIsActive(userID string, isActive bool) (*api.User, error) {
	args := m.Called(userID, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.User), args.Error(1)
}

func (m *MockUserRepository) GetActiveUsersByTeam(teamName string, excludeUserID string) ([]api.User, error) {
	args := m.Called(teamName, excludeUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.User), args.Error(1)
}

// MockPRRepository - мок для PRRepository
type MockPRRepository struct {
	mock.Mock
}

func (m *MockPRRepository) CreatePR(pr *api.PullRequest) (*api.PullRequest, error) {
	args := m.Called(pr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *MockPRRepository) GetPR(prID string) (*api.PullRequest, error) {
	args := m.Called(prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *MockPRRepository) UpdatePRStatus(prID string, status api.PullRequestStatus, mergedAt *time.Time) (*api.PullRequest, error) {
	args := m.Called(prID, status, mergedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *MockPRRepository) GetPRsByReviewer(userID string) ([]api.PullRequestShort, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]api.PullRequestShort), args.Error(1)
}

func (m *MockPRRepository) ReassignReviewer(prID string, oldUserID, newUserID string) (*api.PullRequest, error) {
	args := m.Called(prID, oldUserID, newUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *MockPRRepository) AddReviewer(prID string, userID string) error {
	args := m.Called(prID, userID)
	return args.Error(0)
}

func (m *MockPRRepository) GetReviewerStatistics() ([]storage.ReviewerStatistic, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.ReviewerStatistic), args.Error(1)
}
