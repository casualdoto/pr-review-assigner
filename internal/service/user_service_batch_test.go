package service

import (
	"testing"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/assert"
)

// TestUserService_DeactivateTeamUsers_Success проверяет успешную массовую деактивацию с переназначением
func TestUserService_DeactivateTeamUsers_Success(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)

	userService := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	teamName := "backend"
	userIDsToDeactivate := []string{"u2", "u3"}

	// Мокаем получение команды
	mockTeamRepo.On("GetTeam", teamName).Return(&api.Team{
		TeamName: teamName,
		Members:  []api.TeamMember{},
	}, nil)

	// Мокаем получение всех пользователей команды
	allTeamUsers := []api.User{
		{UserId: "u1", Username: "Alice", TeamName: teamName, IsActive: true},
		{UserId: "u2", Username: "Bob", TeamName: teamName, IsActive: true},
		{UserId: "u3", Username: "Charlie", TeamName: teamName, IsActive: true},
		{UserId: "u4", Username: "David", TeamName: teamName, IsActive: true},
		{UserId: "u5", Username: "Eve", TeamName: teamName, IsActive: true},
	}
	mockUserRepo.On("GetUsersByTeam", teamName).Return(allTeamUsers, nil)

	// Мокаем получение открытых PR
	openPRs := []api.PullRequest{
		{
			PullRequestId:     "pr-1",
			PullRequestName:   "Feature A",
			AuthorId:          "u1",
			Status:            api.PullRequestStatusOPEN,
			AssignedReviewers: []string{"u2", "u3"},
		},
		{
			PullRequestId:     "pr-2",
			PullRequestName:   "Feature B",
			AuthorId:          "u4",
			Status:            api.PullRequestStatusOPEN,
			AssignedReviewers: []string{"u2"},
		},
	}
	mockPRRepo.On("GetOpenPRsByReviewers", userIDsToDeactivate).Return(openPRs, nil)

	// Мокаем массовую деактивацию
	deactivatedUsers := []api.User{
		{UserId: "u2", Username: "Bob", TeamName: teamName, IsActive: false},
		{UserId: "u3", Username: "Charlie", TeamName: teamName, IsActive: false},
	}
	mockUserRepo.On("BatchDeactivateUsers", userIDsToDeactivate).Return(deactivatedUsers, nil)

	// Мокаем batch переназначение - теперь есть достаточно кандидатов
	mockPRRepo.On("BatchReassignReviewers", map[string]map[string]string{
		"pr-1": {"u2": "u4", "u3": "u5"},
		"pr-2": {"u2": "u1"},
	}).Return(nil)

	// Вызываем метод
	result, count, err := userService.DeactivateTeamUsers(teamName, userIDsToDeactivate)

	// Проверяем результат
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 3, count) // 3 переназначения (u2 в pr-1, u3 в pr-1, u2 в pr-2)
	assert.False(t, result[0].IsActive)
	assert.False(t, result[1].IsActive)

	mockTeamRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockPRRepo.AssertExpectations(t)
}

// TestUserService_DeactivateTeamUsers_TeamNotFound проверяет ошибку при несуществующей команде
func TestUserService_DeactivateTeamUsers_TeamNotFound(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)

	userService := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	teamName := "nonexistent"
	userIDs := []string{"u1"}

	mockTeamRepo.On("GetTeam", teamName).Return(nil, storage.ErrNotFound)

	result, count, err := userService.DeactivateTeamUsers(teamName, userIDs)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, count)

	mockTeamRepo.AssertExpectations(t)
}

// TestUserService_DeactivateTeamUsers_UserNotInTeam проверяет ошибку когда пользователь не в команде
func TestUserService_DeactivateTeamUsers_UserNotInTeam(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)

	userService := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	teamName := "backend"
	userIDsToDeactivate := []string{"u2", "u999"}

	mockTeamRepo.On("GetTeam", teamName).Return(&api.Team{
		TeamName: teamName,
	}, nil)

	allTeamUsers := []api.User{
		{UserId: "u1", Username: "Alice", TeamName: teamName, IsActive: true},
		{UserId: "u2", Username: "Bob", TeamName: teamName, IsActive: true},
	}
	mockUserRepo.On("GetUsersByTeam", teamName).Return(allTeamUsers, nil)

	result, count, err := userService.DeactivateTeamUsers(teamName, userIDsToDeactivate)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, count)

	mockTeamRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestUserService_DeactivateTeamUsers_EmptyList проверяет пустой список
func TestUserService_DeactivateTeamUsers_EmptyList(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)

	userService := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	result, count, err := userService.DeactivateTeamUsers("backend", []string{})

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, count)
}

// TestUserService_DeactivateTeamUsers_NoCandidatesAvailable проверяет случай когда нет кандидатов
func TestUserService_DeactivateTeamUsers_NoCandidatesAvailable(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)
	mockPRRepo := new(MockPRRepository)

	userService := NewUserService(mockUserRepo, mockPRRepo, mockTeamRepo)

	teamName := "backend"
	userIDsToDeactivate := []string{"u2", "u3"}

	mockTeamRepo.On("GetTeam", teamName).Return(&api.Team{
		TeamName: teamName,
	}, nil)

	// Только деактивируемые пользователи в команде
	allTeamUsers := []api.User{
		{UserId: "u2", Username: "Bob", TeamName: teamName, IsActive: true},
		{UserId: "u3", Username: "Charlie", TeamName: teamName, IsActive: true},
	}
	mockUserRepo.On("GetUsersByTeam", teamName).Return(allTeamUsers, nil)

	// PR с деактивируемыми ревьюверами
	openPRs := []api.PullRequest{
		{
			PullRequestId:     "pr-1",
			PullRequestName:   "Feature A",
			AuthorId:          "u1",
			Status:            api.PullRequestStatusOPEN,
			AssignedReviewers: []string{"u2", "u3"},
		},
	}
	mockPRRepo.On("GetOpenPRsByReviewers", userIDsToDeactivate).Return(openPRs, nil)

	deactivatedUsers := []api.User{
		{UserId: "u2", Username: "Bob", TeamName: teamName, IsActive: false},
		{UserId: "u3", Username: "Charlie", TeamName: teamName, IsActive: false},
	}
	mockUserRepo.On("BatchDeactivateUsers", userIDsToDeactivate).Return(deactivatedUsers, nil)

	// Переназначение без замены (пустая строка)
	mockPRRepo.On("BatchReassignReviewers", map[string]map[string]string{
		"pr-1": {"u2": "", "u3": ""},
	}).Return(nil)

	result, count, err := userService.DeactivateTeamUsers(teamName, userIDsToDeactivate)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, count)

	mockTeamRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockPRRepo.AssertExpectations(t)
}
