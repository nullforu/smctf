package handlers

import (
	"encoding/json"
	"time"

	"smctf/internal/models"
)

type appConfigResponse struct {
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	HeaderTitle       string    `json:"header_title"`
	HeaderDescription string    `json:"header_description"`
	CTFStartAt        string    `json:"ctf_start_at"`
	CTFEndAt          string    `json:"ctf_end_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type optionalString struct {
	Set   bool
	Value *string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type adminConfigUpdateRequest struct {
	Title             optionalString `json:"title"`
	Description       optionalString `json:"description"`
	HeaderTitle       optionalString `json:"header_title"`
	HeaderDescription optionalString `json:"header_description"`
	CTFStartAt        optionalString `json:"ctf_start_at"`
	CTFEndAt          optionalString `json:"ctf_end_at"`
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
	Title           string  `json:"title" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	Category        string  `json:"category" binding:"required"`
	Points          int     `json:"points" binding:"required"`
	MinimumPoints   *int    `json:"minimum_points"`
	Flag            string  `json:"flag" binding:"required"`
	IsActive        *bool   `json:"is_active"`
	StackEnabled    *bool   `json:"stack_enabled"`
	StackTargetPort *int    `json:"stack_target_port"`
	StackPodSpec    *string `json:"stack_pod_spec"`
}

type updateChallengeRequest struct {
	Title           optionalString `json:"title"`
	Description     optionalString `json:"description"`
	Category        optionalString `json:"category"`
	Points          *int           `json:"points"`
	MinimumPoints   *int           `json:"minimum_points"`
	Flag            optionalString `json:"flag"`
	IsActive        *bool          `json:"is_active"`
	StackEnabled    *bool          `json:"stack_enabled"`
	StackTargetPort *int           `json:"stack_target_port"`
	StackPodSpec    optionalString `json:"stack_pod_spec"`
}

type challengeFileUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
}

type submitRequest struct {
	Flag string `json:"flag" binding:"required"`
}

type createRegistrationKeysRequest struct {
	Count   *int   `json:"count" binding:"required"`
	TeamID  *int64 `json:"team_id" binding:"required"`
	MaxUses *int   `json:"max_uses"`
}

type createTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

type adminMoveUserTeamRequest struct {
	TeamID int64 `json:"team_id" binding:"required"`
}

type adminBlockUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type adminUnblockUserRequest struct {
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
	User         loginUserResponse `json:models.UserRole`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type userMeResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type userDetailResponse struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type adminUserResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type challengeResponse struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Points          int     `json:"points"`
	InitialPoints   int     `json:"initial_points"`
	MinimumPoints   int     `json:"minimum_points"`
	SolveCount      int     `json:"solve_count"`
	IsActive        bool    `json:"is_active"`
	HasFile         bool    `json:"has_file"`
	FileName        *string `json:"file_name,omitempty"`
	StackEnabled    bool    `json:"stack_enabled"`
	StackTargetPort int     `json:"stack_target_port"`
}

type ctfStateResponse struct {
	CTFState string `json:"ctf_state"`
}

type challengesListResponse struct {
	CTFState   string              `json:"ctf_state"`
	Challenges []challengeResponse `json:"challenges,omitempty"`
}

type adminChallengeResponse struct {
	challengeResponse
	StackPodSpec *string `json:"stack_pod_spec,omitempty"`
}

type presignedPostResponse struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type presignedURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	CTFState  string    `json:"ctf_state"`
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

type adminReportChallenge struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Category        string     `json:"category"`
	Points          int        `json:"points"`
	InitialPoints   int        `json:"initial_points"`
	MinimumPoints   int        `json:"minimum_points"`
	SolveCount      int        `json:"solve_count"`
	IsActive        bool       `json:"is_active"`
	FileKey         *string    `json:"file_key,omitempty"`
	FileName        *string    `json:"file_name,omitempty"`
	FileUploadedAt  *time.Time `json:"file_uploaded_at,omitempty"`
	StackEnabled    bool       `json:"stack_enabled"`
	StackTargetPort int        `json:"stack_target_port"`
	StackPodSpec    *string    `json:"stack_pod_spec,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type adminReportUser struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	BlockedReason *string    `json:"blocked_reason,omitempty"`
	BlockedAt     *time.Time `json:"blocked_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type adminReportSubmission struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	ChallengeID  int64     `json:"challenge_id"`
	Correct      bool      `json:"correct"`
	IsFirstBlood bool      `json:"is_first_blood"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

type adminReportResponse struct {
	Challenges       []adminReportChallenge          `json:"challenges"`
	Teams            []models.TeamSummary            `json:"teams"`
	Users            []adminReportUser               `json:"users"`
	Stacks           []models.Stack                  `json:"stacks"`
	RegistrationKeys []models.RegistrationKeySummary `json:"registration_keys"`
	Submissions      []adminReportSubmission         `json:"submissions"`
	AppConfig        []models.AppConfig              `json:"app_config"`
	Timeline         timelineResponse                `json:"timeline"`
	TeamTimeline     teamTimelineResponse            `json:"team_timeline"`
	Leaderboard      models.LeaderboardResponse      `json:"leaderboard"`
	TeamLeaderboard  models.TeamLeaderboardResponse  `json:"team_leaderboard"`
}

type stackResponse struct {
	StackID      string     `json:"stack_id"`
	ChallengeID  int64      `json:"challenge_id"`
	Status       string     `json:"status"`
	NodePublicIP *string    `json:"node_public_ip,omitempty"`
	NodePort     *int       `json:"node_port,omitempty"`
	TargetPort   int        `json:"target_port"`
	TTLExpiresAt *time.Time `json:"ttl_expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CTFState     string     `json:"-"`
}

type stacksListResponse struct {
	CTFState string          `json:"ctf_state"`
	Stacks   []stackResponse `json:"stacks,omitempty"`
}

type adminStackResponse struct {
	StackID           string     `json:"stack_id"`
	TTLExpiresAt      *time.Time `json:"ttl_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UserID            int64      `json:"user_id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	TeamID            int64      `json:"team_id"`
	TeamName          string     `json:"team_name"`
	ChallengeID       int64      `json:"challenge_id"`
	ChallengeTitle    string     `json:"challenge_title"`
	ChallengeCategory string     `json:"challenge_category"`
}

type adminStacksListResponse struct {
	Stacks []adminStackResponse `json:"stacks,omitempty"`
}

func newStackResponse(stack *models.Stack, ctfState string) stackResponse {
	return stackResponse{
		StackID:      stack.StackID,
		ChallengeID:  stack.ChallengeID,
		Status:       stack.Status,
		NodePublicIP: stack.NodePublicIP,
		NodePort:     stack.NodePort,
		TargetPort:   stack.TargetPort,
		TTLExpiresAt: stack.TTLExpiresAt,
		CreatedAt:    stack.CreatedAt.UTC(),
		UpdatedAt:    stack.UpdatedAt.UTC(),
		CTFState:     ctfState,
	}
}

func newAdminStackResponse(stack models.AdminStackSummary) adminStackResponse {
	return adminStackResponse{
		StackID:           stack.StackID,
		TTLExpiresAt:      timePtrUTC(stack.TTLExpiresAt),
		CreatedAt:         stack.CreatedAt.UTC(),
		UpdatedAt:         stack.UpdatedAt.UTC(),
		UserID:            stack.UserID,
		Username:          stack.Username,
		Email:             stack.Email,
		TeamID:            stack.TeamID,
		TeamName:          stack.TeamName,
		ChallengeID:       stack.ChallengeID,
		ChallengeTitle:    stack.ChallengeTitle,
		ChallengeCategory: stack.ChallengeCategory,
	}
}

func newAdminReportChallenge(challenge models.Challenge) adminReportChallenge {
	return adminReportChallenge{
		ID:              challenge.ID,
		Title:           challenge.Title,
		Description:     challenge.Description,
		Category:        challenge.Category,
		Points:          challenge.Points,
		InitialPoints:   challenge.InitialPoints,
		MinimumPoints:   challenge.MinimumPoints,
		SolveCount:      challenge.SolveCount,
		IsActive:        challenge.IsActive,
		FileKey:         challenge.FileKey,
		FileName:        challenge.FileName,
		FileUploadedAt:  challenge.FileUploadedAt,
		StackEnabled:    challenge.StackEnabled,
		StackTargetPort: challenge.StackTargetPort,
		StackPodSpec:    challenge.StackPodSpec,
		CreatedAt:       challenge.CreatedAt.UTC(),
	}
}

func newAdminReportUser(user models.User) adminReportUser {
	return adminReportUser{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		TeamID:        user.TeamID,
		TeamName:      user.TeamName,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
		CreatedAt:     user.CreatedAt.UTC(),
		UpdatedAt:     user.UpdatedAt.UTC(),
	}
}

func newAdminReportSubmission(sub models.Submission) adminReportSubmission {
	return adminReportSubmission{
		ID:           sub.ID,
		UserID:       sub.UserID,
		ChallengeID:  sub.ChallengeID,
		Correct:      sub.Correct,
		IsFirstBlood: sub.IsFirstBlood,
		SubmittedAt:  sub.SubmittedAt.UTC(),
	}
}

func timePtrUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func newUserMeResponse(user *models.User) userMeResponse {
	return userMeResponse{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		TeamID:        user.TeamID,
		TeamName:      user.TeamName,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newUserDetailResponse(user *models.User) userDetailResponse {
	return userDetailResponse{
		ID:            user.ID,
		Username:      user.Username,
		Role:          user.Role,
		TeamID:        user.TeamID,
		TeamName:      user.TeamName,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newAdminUserResponse(user *models.User) adminUserResponse {
	return adminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		TeamID:        user.TeamID,
		TeamName:      user.TeamName,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newChallengeResponse(challenge *models.Challenge) challengeResponse {
	hasFile := challenge.FileKey != nil && *challenge.FileKey != ""
	return challengeResponse{
		ID:              challenge.ID,
		Title:           challenge.Title,
		Description:     challenge.Description,
		Category:        challenge.Category,
		Points:          challenge.Points,
		InitialPoints:   challenge.InitialPoints,
		MinimumPoints:   challenge.MinimumPoints,
		SolveCount:      challenge.SolveCount,
		IsActive:        challenge.IsActive,
		HasFile:         hasFile,
		FileName:        challenge.FileName,
		StackEnabled:    challenge.StackEnabled,
		StackTargetPort: challenge.StackTargetPort,
	}
}

func newTeamResponse(team *models.Team) teamResponse {
	return teamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedAt: team.CreatedAt,
	}
}
