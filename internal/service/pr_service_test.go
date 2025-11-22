package service

import (
	"testing"
	"time"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPRService_CreatePR_Success(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	author := &api.User{
		UserId:   "u1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: true,
	}

	candidates := []api.User{
		{UserId: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{UserId: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	}

	expectedPR := &api.PullRequest{
		PullRequestId:     "pr-1",
		PullRequestName:   "Test PR",
		AuthorId:          "u1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u2", "u3"},
		CreatedAt:         func() *time.Time { t := time.Now(); return &t }(),
	}

	mockUserRepo.On("GetUser", "u1").Return(author, nil)
	mockPRRepo.On("GetPR", "pr-1").Return(nil, storage.ErrNotFound).Once()
	mockUserRepo.On("GetActiveUsersByTeam", "backend", "u1").Return(candidates, nil)
	mockPRRepo.On("CreatePR", mock.AnythingOfType("*api.PullRequest")).Return(expectedPR, nil)

	result, err := service.CreatePR("pr-1", "Test PR", "u1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "pr-1", result.PullRequestId)
	assert.Equal(t, api.PullRequestStatusOPEN, result.Status)
	assert.LessOrEqual(t, len(result.AssignedReviewers), 2)
	mockPRRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestPRService_CreatePR_AuthorNotFound(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	mockUserRepo.On("GetUser", "u1").Return(nil, storage.ErrNotFound)

	result, err := service.CreatePR("pr-1", "Test PR", "u1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockUserRepo.AssertExpectations(t)
}

func TestPRService_CreatePR_AlreadyExists(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	author := &api.User{
		UserId:   "u1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: true,
	}

	existingPR := &api.PullRequest{
		PullRequestId: "pr-1",
		Status:        api.PullRequestStatusOPEN,
	}

	mockUserRepo.On("GetUser", "u1").Return(author, nil)
	mockPRRepo.On("GetPR", "pr-1").Return(existingPR, nil)

	result, err := service.CreatePR("pr-1", "Test PR", "u1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrPRExists, err)
	mockPRRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestPRService_MergePR_Success(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	pr := &api.PullRequest{
		PullRequestId: "pr-1",
		Status:        api.PullRequestStatusOPEN,
	}

	mergedPR := &api.PullRequest{
		PullRequestId: "pr-1",
		Status:        api.PullRequestStatusMERGED,
		MergedAt:      func() *time.Time { t := time.Now(); return &t }(),
	}

	mockPRRepo.On("GetPR", "pr-1").Return(pr, nil).Once()
	mockPRRepo.On("UpdatePRStatus", "pr-1", api.PullRequestStatusMERGED, mock.AnythingOfType("*time.Time")).Return(mergedPR, nil)

	result, err := service.MergePR("pr-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, api.PullRequestStatusMERGED, result.Status)
	assert.NotNil(t, result.MergedAt)
	mockPRRepo.AssertExpectations(t)
}

func TestPRService_MergePR_Idempotent(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	mergedPR := &api.PullRequest{
		PullRequestId: "pr-1",
		Status:        api.PullRequestStatusMERGED,
		MergedAt:      func() *time.Time { t := time.Now(); return &t }(),
	}

	mockPRRepo.On("GetPR", "pr-1").Return(mergedPR, nil)

	result, err := service.MergePR("pr-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, api.PullRequestStatusMERGED, result.Status)
	mockPRRepo.AssertExpectations(t)
	mockPRRepo.AssertNotCalled(t, "UpdatePRStatus")
}

func TestPRService_MergePR_NotFound(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	mockPRRepo.On("GetPR", "pr-1").Return(nil, storage.ErrNotFound)

	result, err := service.MergePR("pr-1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockPRRepo.AssertExpectations(t)
}

func TestPRService_ReassignReviewer_Success(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	pr := &api.PullRequest{
		PullRequestId:     "pr-1",
		AuthorId:          "u1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u2", "u3"},
	}

	oldReviewer := &api.User{
		UserId:   "u2",
		Username: "Bob",
		TeamName: "backend",
		IsActive: true,
	}

	candidates := []api.User{
		{UserId: "u4", Username: "David", TeamName: "backend", IsActive: true},
		{UserId: "u5", Username: "Eve", TeamName: "backend", IsActive: true},
	}

	updatedPR := &api.PullRequest{
		PullRequestId:     "pr-1",
		AuthorId:          "u1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u3", "u4"},
	}

	mockPRRepo.On("GetPR", "pr-1").Return(pr, nil).Once()
	mockUserRepo.On("GetUser", "u2").Return(oldReviewer, nil)
	mockUserRepo.On("GetActiveUsersByTeam", "backend", "u2").Return(candidates, nil)
	mockPRRepo.On("ReassignReviewer", "pr-1", "u2", mock.AnythingOfType("string")).Return(updatedPR, nil)

	result, newUserID, err := service.ReassignReviewer("pr-1", "u2")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, newUserID)
	assert.NotEqual(t, "u2", newUserID)
	mockPRRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestPRService_ReassignReviewer_PRMerged(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	pr := &api.PullRequest{
		PullRequestId: "pr-1",
		Status:        api.PullRequestStatusMERGED,
	}

	mockPRRepo.On("GetPR", "pr-1").Return(pr, nil)

	result, newUserID, err := service.ReassignReviewer("pr-1", "u2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newUserID)
	assert.Equal(t, ErrPRMerged, err)
	mockPRRepo.AssertExpectations(t)
}

func TestPRService_ReassignReviewer_NotAssigned(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	pr := &api.PullRequest{
		PullRequestId:     "pr-1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"u3"},
	}

	mockPRRepo.On("GetPR", "pr-1").Return(pr, nil)

	result, newUserID, err := service.ReassignReviewer("pr-1", "u2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, newUserID)
	assert.Equal(t, ErrNotAssigned, err)
	mockPRRepo.AssertExpectations(t)
}

func TestPRService_GetPRsByReviewer_Success(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	user := &api.User{
		UserId:   "u2",
		Username: "Bob",
		TeamName: "backend",
		IsActive: true,
	}

	prs := []api.PullRequestShort{
		{PullRequestId: "pr-1", PullRequestName: "PR 1", AuthorId: "u1", Status: api.PullRequestShortStatusOPEN},
		{PullRequestId: "pr-2", PullRequestName: "PR 2", AuthorId: "u1", Status: api.PullRequestShortStatusOPEN},
	}

	mockUserRepo.On("GetUser", "u2").Return(user, nil)
	mockPRRepo.On("GetPRsByReviewer", "u2").Return(prs, nil)

	result, err := service.GetPRsByReviewer("u2")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	mockPRRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestPRService_GetPRsByReviewer_UserNotFound(t *testing.T) {
	mockPRRepo := new(MockPRRepository)
	mockUserRepo := new(MockUserRepository)
	mockTeamRepo := new(MockTeamRepository)

	service := NewPRService(mockPRRepo, mockUserRepo, mockTeamRepo)

	mockUserRepo.On("GetUser", "u2").Return(nil, storage.ErrNotFound)

	result, err := service.GetPRsByReviewer("u2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockUserRepo.AssertExpectations(t)
}
