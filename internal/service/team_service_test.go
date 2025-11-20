package service

import (
	"testing"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamService_CreateOrUpdateTeam_Success(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewTeamService(mockTeamRepo, mockUserRepo)

	team := &api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
		},
	}

	expectedTeam := &api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
		},
	}

	mockTeamRepo.On("TeamExists", "backend").Return(false, nil)
	mockTeamRepo.On("CreateTeam", "backend").Return(nil)
	mockUserRepo.On("CreateOrUpdateUser", mock.AnythingOfType("*api.User")).Return(nil).Times(2)
	mockTeamRepo.On("GetTeam", "backend").Return(expectedTeam, nil)

	result, err := service.CreateOrUpdateTeam(team)

	assert.NoError(t, err)
	assert.Equal(t, expectedTeam, result)
	mockTeamRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestTeamService_CreateOrUpdateTeam_TeamExists(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewTeamService(mockTeamRepo, mockUserRepo)

	team := &api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
		},
	}

	mockTeamRepo.On("TeamExists", "backend").Return(true, nil)

	result, err := service.CreateOrUpdateTeam(team)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrTeamExists, err)
	mockTeamRepo.AssertExpectations(t)
}

func TestTeamService_GetTeam_Success(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewTeamService(mockTeamRepo, mockUserRepo)

	expectedTeam := &api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
		},
	}

	mockTeamRepo.On("GetTeam", "backend").Return(expectedTeam, nil)

	result, err := service.GetTeam("backend")

	assert.NoError(t, err)
	assert.Equal(t, expectedTeam, result)
	mockTeamRepo.AssertExpectations(t)
}

func TestTeamService_GetTeam_NotFound(t *testing.T) {
	mockTeamRepo := new(MockTeamRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewTeamService(mockTeamRepo, mockUserRepo)

	mockTeamRepo.On("GetTeam", "backend").Return(nil, storage.ErrNotFound)

	result, err := service.GetTeam("backend")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockTeamRepo.AssertExpectations(t)
}

