package service

import (
	"pr-review-assigner/internal/api"
)

// Типизированные ошибки для маппинга на API коды
var (
	ErrTeamExists  = &ServiceError{Code: api.TEAMEXISTS, Message: "team_name already exists"}
	ErrPRExists    = &ServiceError{Code: api.PREXISTS, Message: "PR id already exists"}
	ErrPRMerged    = &ServiceError{Code: api.PRMERGED, Message: "cannot reassign on merged PR"}
	ErrNotAssigned = &ServiceError{Code: api.NOTASSIGNED, Message: "reviewer is not assigned to this PR"}
	ErrNoCandidate = &ServiceError{Code: api.NOCANDIDATE, Message: "no active replacement candidate in team"}
	ErrNotFound    = &ServiceError{Code: api.NOTFOUND, Message: "resource not found"}
)

// ServiceError представляет ошибку сервисного слоя с кодом API
type ServiceError struct {
	Code    api.ErrorResponseErrorCode
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// IsServiceError проверяет, является ли ошибка ServiceError
func IsServiceError(err error) bool {
	_, ok := err.(*ServiceError)
	return ok
}

// GetServiceError возвращает ServiceError из ошибки, если это возможно
func GetServiceError(err error) *ServiceError {
	if se, ok := err.(*ServiceError); ok {
		return se
	}
	return nil
}
