package service

import (
	"testing"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestUserService_SetUserIsActive_Success_Activate(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	expectedUser := &api.User{
		UserId:   "u1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: true,
	}

	mockUserRepo.On("UpdateUserIsActive", "u1", true).Return(expectedUser, nil)

	result, err := service.SetUserIsActive("u1", true)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, result)
	assert.True(t, result.IsActive)
	mockUserRepo.AssertExpectations(t)
}

func TestUserService_SetUserIsActive_Deactivate_WithReassignment(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	deactivatedUser := &api.User{
		UserId:   "u2",
		Username: "Bob",
		TeamName: "backend",
		IsActive: false,
	}

	// PR где u2 является ревьювером
	prs := []api.PullRequestShort{
		{PullRequestId: "pr-1", PullRequestName: "Test PR", AuthorId: "u1", Status: api.PullRequestShortStatusOPEN},
	}

	fullPR := &api.PullRequest{
		PullRequestId:     "pr-1",
		AuthorId:          "u1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u2", "u3"},
	}

	candidates := []api.User{
		{UserId: "u4", Username: "David", TeamName: "backend", IsActive: true},
	}

	updatedPR := &api.PullRequest{
		PullRequestId:     "pr-1",
		AuthorId:          "u1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u3", "u4"},
	}

	mockUserRepo.On("UpdateUserIsActive", "u2", false).Return(deactivatedUser, nil)
	mockPRRepo.On("GetPRsByReviewer", "u2").Return(prs, nil)
	mockPRRepo.On("GetPR", "pr-1").Return(fullPR, nil)
	mockUserRepo.On("GetActiveUsersByTeam", "backend", "u2").Return(candidates, nil)
	mockPRRepo.On("ReassignReviewer", "pr-1", "u2", "u4").Return(updatedPR, nil)

	result, err := service.SetUserIsActive("u2", false)

	assert.NoError(t, err)
	assert.Equal(t, deactivatedUser, result)
	assert.False(t, result.IsActive)
	mockUserRepo.AssertExpectations(t)
	mockPRRepo.AssertExpectations(t)
}

func TestUserService_SetUserIsActive_Deactivate_NoPRs(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	deactivatedUser := &api.User{
		UserId:   "u2",
		Username: "Bob",
		TeamName: "backend",
		IsActive: false,
	}

	mockUserRepo.On("UpdateUserIsActive", "u2", false).Return(deactivatedUser, nil)
	mockPRRepo.On("GetPRsByReviewer", "u2").Return([]api.PullRequestShort{}, nil)

	result, err := service.SetUserIsActive("u2", false)

	assert.NoError(t, err)
	assert.Equal(t, deactivatedUser, result)
	assert.False(t, result.IsActive)
	mockUserRepo.AssertExpectations(t)
	mockPRRepo.AssertExpectations(t)
}

func TestUserService_SetUserIsActive_UserNotFound(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	mockUserRepo.On("UpdateUserIsActive", "u1", false).Return(nil, storage.ErrNotFound)

	result, err := service.SetUserIsActive("u1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockUserRepo.AssertExpectations(t)
}
