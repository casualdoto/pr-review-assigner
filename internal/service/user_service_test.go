package service

import (
	"testing"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestUserService_SetUserIsActive_Success(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	service := NewUserService(mockUserRepo)

	expectedUser := &api.User{
		UserId:   "u1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: false,
	}

	mockUserRepo.On("UpdateUserIsActive", "u1", false).Return(expectedUser, nil)

	result, err := service.SetUserIsActive("u1", false)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, result)
	assert.False(t, result.IsActive)
	mockUserRepo.AssertExpectations(t)
}

func TestUserService_SetUserIsActive_UserNotFound(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	service := NewUserService(mockUserRepo)

	mockUserRepo.On("UpdateUserIsActive", "u1", false).Return(nil, storage.ErrNotFound)

	result, err := service.SetUserIsActive("u1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	mockUserRepo.AssertExpectations(t)
}

