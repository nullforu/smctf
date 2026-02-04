package handlers

import (
	"time"

	"smctf/internal/models"
)

type appConfigResponse struct {
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	HeaderTitle       string    `json:"header_title"`
	HeaderDescription string    `json:"header_description"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type adminConfigUpdateRequest struct {
	Title             *string `json:"title"`
	Description       *string `json:"description"`
	HeaderTitle       *string `json:"header_title"`
	HeaderDescription *string `json:"header_description"`
}

type meUpdateRequest struct {
	Username *string `json:"username"`
}

type registerRequest struct {
	Email           string `json:"email" binding:"required"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	RegistrationKey string `json:"registration_key" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type createChallengeRequest struct {
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description" binding:"required"`
	Category      string `json:"category" binding:"required"`
	Points        int    `json:"points" binding:"required"`
	MinimumPoints *int   `json:"minimum_points"`
	Flag          string `json:"flag" binding:"required"`
	IsActive      *bool  `json:"is_active"`
}

type updateChallengeRequest struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	Category      *string `json:"category"`
	Points        *int    `json:"points"`
	MinimumPoints *int    `json:"minimum_points"`
	Flag          *string `json:"flag"`
	IsActive      *bool   `json:"is_active"`
}

type challengeFileUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
}

type submitRequest struct {
	Flag string `json:"flag" binding:"required"`
}

type createRegistrationKeysRequest struct {
	Count  *int   `json:"count" binding:"required"`
	TeamID *int64 `json:"team_id" binding:"required"`
}

type createTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

type registerResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type loginUserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type loginResponse struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	User         loginUserResponse `json:"user"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type userMeResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TeamID   int64  `json:"team_id"`
	TeamName string `json:"team_name"`
}

type userDetailResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TeamID   int64  `json:"team_id"`
	TeamName string `json:"team_name"`
}

type challengeResponse struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Points        int     `json:"points"`
	InitialPoints int     `json:"initial_points"`
	MinimumPoints int     `json:"minimum_points"`
	SolveCount    int     `json:"solve_count"`
	IsActive      bool    `json:"is_active"`
	HasFile       bool    `json:"has_file"`
	FileName      *string `json:"file_name,omitempty"`
}

type presignedPostResponse struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type presignedURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type challengeFileUploadResponse struct {
	Challenge challengeResponse     `json:"challenge"`
	Upload    presignedPostResponse `json:"upload"`
}

type teamResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type timelineResponse struct {
	Submissions []models.TimelineSubmission `json:"submissions"`
}

type teamTimelineResponse struct {
	Submissions []models.TeamTimelineSubmission `json:"submissions"`
}

func newUserMeResponse(user *models.User) userMeResponse {
	return userMeResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
		TeamID:   user.TeamID,
		TeamName: user.TeamName,
	}
}

func newUserDetailResponse(user *models.User) userDetailResponse {
	return userDetailResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		TeamID:   user.TeamID,
		TeamName: user.TeamName,
	}
}

func newChallengeResponse(challenge *models.Challenge) challengeResponse {
	hasFile := challenge.FileKey != nil && *challenge.FileKey != ""
	return challengeResponse{
		ID:            challenge.ID,
		Title:         challenge.Title,
		Description:   challenge.Description,
		Category:      challenge.Category,
		Points:        challenge.Points,
		InitialPoints: challenge.InitialPoints,
		MinimumPoints: challenge.MinimumPoints,
		SolveCount:    challenge.SolveCount,
		IsActive:      challenge.IsActive,
		HasFile:       hasFile,
		FileName:      challenge.FileName,
	}
}

func newTeamResponse(team *models.Team) teamResponse {
	return teamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedAt: team.CreatedAt,
	}
}
