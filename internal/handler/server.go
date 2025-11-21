package handler

import (
	"encoding/json"
	"net/http"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/service"
)

// Server реализует ServerInterface для обработки HTTP запросов
type Server struct {
	teamService *service.TeamService
	userService *service.UserService
	prService   *service.PRService
}

// NewServer создает новый экземпляр сервера
func NewServer(teamService *service.TeamService, userService *service.UserService, prService *service.PRService) *Server {
	return &Server{
		teamService: teamService,
		userService: userService,
		prService:   prService,
	}
}

// writeJSON записывает JSON ответ
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError записывает ошибку в формате ErrorResponse
func (s *Server) writeError(w http.ResponseWriter, statusCode int, code api.ErrorResponseErrorCode, message string) {
	errorResponse := api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode `json:"code"`
			Message string                     `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
	s.writeJSON(w, statusCode, errorResponse)
}

// handleServiceError обрабатывает ошибку сервиса и возвращает соответствующий HTTP ответ
func (s *Server) handleServiceError(w http.ResponseWriter, err error) {
	if service.IsServiceError(err) {
		se := service.GetServiceError(err)
		switch se.Code {
		case api.TEAMEXISTS:
			s.writeError(w, http.StatusBadRequest, se.Code, se.Message)
		case api.PRMERGED, api.NOTASSIGNED, api.NOCANDIDATE, api.PREXISTS:
			s.writeError(w, http.StatusConflict, se.Code, se.Message)
		case api.NOTFOUND:
			s.writeError(w, http.StatusNotFound, se.Code, se.Message)
		default:
			s.writeError(w, http.StatusInternalServerError, api.NOTFOUND, "internal server error")
		}
		return
	}
	// Неизвестная ошибка
	s.writeError(w, http.StatusInternalServerError, api.NOTFOUND, "internal server error")
}

// PostTeamAdd создает команду с участниками
// (POST /team/add)
func (s *Server) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	var team api.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		s.writeError(w, http.StatusBadRequest, api.NOTFOUND, "invalid request body")
		return
	}

	result, err := s.teamService.CreateOrUpdateTeam(&team)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		Team *api.Team `json:"team"`
	}{
		Team: result,
	}
	s.writeJSON(w, http.StatusCreated, response)
}

// GetTeamGet получает команду с участниками
// (GET /team/get)
func (s *Server) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	team, err := s.teamService.GetTeam(params.TeamName)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	s.writeJSON(w, http.StatusOK, team)
}

// PostUsersSetIsActive устанавливает флаг активности пользователя
// (POST /users/setIsActive)
func (s *Server) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	var req api.PostUsersSetIsActiveJSONBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, api.NOTFOUND, "invalid request body")
		return
	}

	user, err := s.userService.SetUserIsActive(req.UserId, req.IsActive)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		User *api.User `json:"user"`
	}{
		User: user,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// PostPullRequestCreate создает PR и автоматически назначает до 2 ревьюверов из команды автора
// (POST /pullRequest/create)
func (s *Server) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	var req api.PostPullRequestCreateJSONBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, api.NOTFOUND, "invalid request body")
		return
	}

	pr, err := s.prService.CreatePR(req.PullRequestId, req.PullRequestName, req.AuthorId)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		PR *api.PullRequest `json:"pr"`
	}{
		PR: pr,
	}
	s.writeJSON(w, http.StatusCreated, response)
}

// PostPullRequestMerge помечает PR как MERGED (идемпотентная операция)
// (POST /pullRequest/merge)
func (s *Server) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	var req api.PostPullRequestMergeJSONBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, api.NOTFOUND, "invalid request body")
		return
	}

	pr, err := s.prService.MergePR(req.PullRequestId)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		PR *api.PullRequest `json:"pr"`
	}{
		PR: pr,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// PostPullRequestReassign переназначает конкретного ревьювера на другого из его команды
// (POST /pullRequest/reassign)
func (s *Server) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	var req api.PostPullRequestReassignJSONBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, api.NOTFOUND, "invalid request body")
		return
	}

	pr, newUserID, err := s.prService.ReassignReviewer(req.PullRequestId, req.OldUserId)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		PR         *api.PullRequest `json:"pr"`
		ReplacedBy string           `json:"replaced_by"`
	}{
		PR:         pr,
		ReplacedBy: newUserID,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// GetUsersGetReview получает PR'ы, где пользователь назначен ревьювером
// (GET /users/getReview)
func (s *Server) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	prs, err := s.prService.GetPRsByReviewer(params.UserId)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}

	response := struct {
		UserId       string                 `json:"user_id"`
		PullRequests []api.PullRequestShort `json:"pull_requests"`
	}{
		UserId:       params.UserId,
		PullRequests: prs,
	}
	s.writeJSON(w, http.StatusOK, response)
}
