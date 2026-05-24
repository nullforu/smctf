package handlers

import (
	"encoding/json"
	"time"

	"smctf/internal/models"
	vmpkg "smctf/internal/vm"
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

type optionalInt64 struct {
	Set   bool
	Value *int64
}

func (o *optionalInt64) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var value int64
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

type createChallengeRequest struct {
	Title               string  `json:"title" binding:"required"`
	Description         string  `json:"description" binding:"required"`
	Category            string  `json:"category" binding:"required"`
	Points              int     `json:"points" binding:"required"`
	MinimumPoints       *int    `json:"minimum_points"`
	Flag                string  `json:"flag" binding:"required"`
	PreviousChallengeID *int64  `json:"previous_challenge_id"`
	IsActive            *bool   `json:"is_active"`
	VMEnabled           *bool   `json:"vm_enabled"`
	VMSpec              *string `json:"vm_spec"`
}

type updateChallengeRequest struct {
	Title               optionalString `json:"title"`
	Description         optionalString `json:"description"`
	Category            optionalString `json:"category"`
	Points              *int           `json:"points"`
	MinimumPoints       *int           `json:"minimum_points"`
	Flag                optionalString `json:"flag"`
	PreviousChallengeID optionalInt64  `json:"previous_challenge_id"`
	IsActive            *bool          `json:"is_active"`
	VMEnabled           *bool          `json:"vm_enabled"`
	VMSpec              optionalString `json:"vm_spec"`
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
	Name       string `json:"name" binding:"required"`
	DivisionID int64  `json:"division_id" binding:"required"`
}

type adminMoveUserTeamRequest struct {
	TeamID int64 `json:"team_id" binding:"required"`
}

type adminBlockUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type adminUnblockUserRequest struct{}

type registerResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type loginUserResponse = userMeResponse

type loginResponse struct {
	User loginUserResponse `json:"user"`
}

type refreshResponse struct {
	Status string `json:"status"`
}

type userMeResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	DivisionID    int64      `json:"division_id"`
	DivisionName  string     `json:"division_name"`
	VMCount       int        `json:"vm_count"`
	VMLimit       int        `json:"vm_limit"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type userDetailResponse struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	DivisionID    int64      `json:"division_id"`
	DivisionName  string     `json:"division_name"`
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
	DivisionID    int64      `json:"division_id"`
	DivisionName  string     `json:"division_name"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type challengeResponse struct {
	ID                  int64   `json:"id"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	Category            string  `json:"category"`
	Points              int     `json:"points"`
	InitialPoints       int     `json:"initial_points"`
	MinimumPoints       int     `json:"minimum_points"`
	SolveCount          int     `json:"solve_count"`
	PreviousChallengeID *int64  `json:"previous_challenge_id,omitempty"`
	IsActive            bool    `json:"is_active"`
	IsLocked            bool    `json:"is_locked"`
	HasFile             bool    `json:"has_file"`
	FileName            *string `json:"file_name,omitempty"`
	VMEnabled           bool    `json:"vm_enabled"`
}

type lockedChallengeResponse struct {
	ID                        int64   `json:"id"`
	Title                     string  `json:"title"`
	Category                  string  `json:"category"`
	Points                    int     `json:"points"`
	InitialPoints             int     `json:"initial_points"`
	MinimumPoints             int     `json:"minimum_points"`
	SolveCount                int     `json:"solve_count"`
	PreviousChallengeID       *int64  `json:"previous_challenge_id,omitempty"`
	PreviousChallengeTitle    *string `json:"previous_challenge_title,omitempty"`
	PreviousChallengeCategory *string `json:"previous_challenge_category,omitempty"`
	IsActive                  bool    `json:"is_active"`
	IsLocked                  bool    `json:"is_locked"`
}

type ctfStateResponse struct {
	CTFState string `json:"ctf_state"`
}

type challengesListResponse struct {
	CTFState   string `json:"ctf_state"`
	Challenges []any  `json:"challenges,omitempty"`
}

type adminChallengeResponse struct {
	challengeResponse
	VMSpec *string `json:"vm_spec,omitempty"`
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

type challengeFileDownloadResponse struct {
	Download presignedURLResponse `json:"download"`
}

type teamResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	DivisionID int64     `json:"division_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type createDivisionRequest struct {
	Name string `json:"name" binding:"required"`
}

type timelineResponse struct {
	Submissions []models.TimelineSubmission `json:"submissions"`
}

type teamTimelineResponse struct {
	Submissions []models.TeamTimelineSubmission `json:"submissions"`
}

type adminReportChallenge struct {
	ID                  int64      `json:"id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Category            string     `json:"category"`
	Points              int        `json:"points"`
	InitialPoints       int        `json:"initial_points"`
	MinimumPoints       int        `json:"minimum_points"`
	SolveCount          int        `json:"solve_count"`
	PreviousChallengeID *int64     `json:"previous_challenge_id,omitempty"`
	IsActive            bool       `json:"is_active"`
	FileKey             *string    `json:"file_key,omitempty"`
	FileName            *string    `json:"file_name,omitempty"`
	FileUploadedAt      *time.Time `json:"file_uploaded_at,omitempty"`
	VMEnabled           bool       `json:"vm_enabled"`
	VMSpec              *string    `json:"vm_spec,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type adminReportUser struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	TeamID        int64      `json:"team_id"`
	TeamName      string     `json:"team_name"`
	DivisionID    int64      `json:"division_id"`
	DivisionName  string     `json:"division_name"`
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
	Divisions        []models.Division               `json:"divisions"`
	Teams            []models.TeamSummary            `json:"teams"`
	Users            []adminReportUser               `json:"users"`
	VMs              []models.VM                     `json:"vms"`
	RegistrationKeys []models.RegistrationKeySummary `json:"registration_keys"`
	Submissions      []adminReportSubmission         `json:"submissions"`
	AppConfig        []models.AppConfig              `json:"app_config"`
	Timeline         timelineResponse                `json:"timeline"`
	TeamTimeline     teamTimelineResponse            `json:"team_timeline"`
	Leaderboard      models.LeaderboardResponse      `json:"leaderboard"`
	TeamLeaderboard  models.TeamLeaderboardResponse  `json:"team_leaderboard"`
}

type vmResponse struct {
	VMID              string              `json:"vm_id"`
	ChallengeID       int64               `json:"challenge_id"`
	Status            string              `json:"status"`
	NodeName          *string             `json:"node_name,omitempty"`
	ExternalIP        *string             `json:"external_ip,omitempty"`
	Ports             []vmpkg.PortMapping `json:"ports,omitempty"`
	TTLExpiresAt      *time.Time          `json:"ttl_expires_at,omitempty"`
	LastError         *string             `json:"last_error,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	CreatedByUserID   int64               `json:"created_by_user_id"`
	CreatedByUsername string              `json:"created_by_username"`
	ChallengeTitle    string              `json:"challenge_title"`
	CTFState          string              `json:"-"`
}

type vmsListResponse struct {
	CTFState string       `json:"ctf_state"`
	VMs      []vmResponse `json:"vms,omitempty"`
}

type adminVMResponse struct {
	VMID              string     `json:"vm_id"`
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

type adminVMsListResponse struct {
	VMs []adminVMResponse `json:"vms,omitempty"`
}

func newVMResponse(vm *models.VM, ctfState string) vmResponse {
	return vmResponse{
		VMID:              vm.VMID,
		ChallengeID:       vm.ChallengeID,
		Status:            vm.Status,
		NodeName:          vm.NodeName,
		ExternalIP:        vm.ExternalIP,
		Ports:             []vmpkg.PortMapping(vm.Ports),
		TTLExpiresAt:      vm.TTLExpiresAt,
		LastError:         vm.LastError,
		CreatedAt:         vm.CreatedAt.UTC(),
		UpdatedAt:         vm.UpdatedAt.UTC(),
		CreatedByUserID:   vm.UserID,
		CreatedByUsername: vm.Username,
		ChallengeTitle:    vm.ChallengeTitle,
		CTFState:          ctfState,
	}
}

func newAdminVMResponse(vm models.AdminVMSummary) adminVMResponse {
	return adminVMResponse{
		VMID:              vm.VMID,
		TTLExpiresAt:      timePtrUTC(vm.TTLExpiresAt),
		CreatedAt:         vm.CreatedAt.UTC(),
		UpdatedAt:         vm.UpdatedAt.UTC(),
		UserID:            vm.UserID,
		Username:          vm.Username,
		Email:             vm.Email,
		TeamID:            vm.TeamID,
		TeamName:          vm.TeamName,
		ChallengeID:       vm.ChallengeID,
		ChallengeTitle:    vm.ChallengeTitle,
		ChallengeCategory: vm.ChallengeCategory,
	}
}

func newAdminReportChallenge(challenge models.Challenge) adminReportChallenge {
	return adminReportChallenge{
		ID:                  challenge.ID,
		Title:               challenge.Title,
		Description:         challenge.Description,
		Category:            challenge.Category,
		Points:              challenge.Points,
		InitialPoints:       challenge.InitialPoints,
		MinimumPoints:       challenge.MinimumPoints,
		SolveCount:          challenge.SolveCount,
		PreviousChallengeID: challenge.PreviousChallengeID,
		IsActive:            challenge.IsActive,
		FileKey:             challenge.FileKey,
		FileName:            challenge.FileName,
		FileUploadedAt:      challenge.FileUploadedAt,
		VMEnabled:           challenge.VMEnabled,
		VMSpec:              challenge.VMSpec,
		CreatedAt:           challenge.CreatedAt.UTC(),
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
		DivisionID:    user.DivisionID,
		DivisionName:  user.DivisionName,
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

func newUserMeResponse(user *models.User, vmCount, vmLimit int) userMeResponse {
	return userMeResponse{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		TeamID:        user.TeamID,
		TeamName:      user.TeamName,
		DivisionID:    user.DivisionID,
		DivisionName:  user.DivisionName,
		VMCount:       vmCount,
		VMLimit:       vmLimit,
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
		DivisionID:    user.DivisionID,
		DivisionName:  user.DivisionName,
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
		DivisionID:    user.DivisionID,
		DivisionName:  user.DivisionName,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newChallengeResponse(challenge *models.Challenge) challengeResponse {
	hasFile := challenge.FileKey != nil && *challenge.FileKey != ""
	return challengeResponse{
		ID:                  challenge.ID,
		Title:               challenge.Title,
		Description:         challenge.Description,
		Category:            challenge.Category,
		Points:              challenge.Points,
		InitialPoints:       challenge.InitialPoints,
		MinimumPoints:       challenge.MinimumPoints,
		SolveCount:          challenge.SolveCount,
		PreviousChallengeID: challenge.PreviousChallengeID,
		IsActive:            challenge.IsActive,
		IsLocked:            false,
		HasFile:             hasFile,
		FileName:            challenge.FileName,
		VMEnabled:           challenge.VMEnabled,
	}
}

func newLockedChallengeResponse(challenge *models.Challenge, previous *models.Challenge) lockedChallengeResponse {
	var prevTitle *string
	var prevCategory *string
	if previous != nil {
		prevTitle = &previous.Title
		prevCategory = &previous.Category
	}
	return lockedChallengeResponse{
		ID:                        challenge.ID,
		Title:                     challenge.Title,
		Category:                  challenge.Category,
		Points:                    challenge.Points,
		InitialPoints:             challenge.InitialPoints,
		MinimumPoints:             challenge.MinimumPoints,
		SolveCount:                challenge.SolveCount,
		PreviousChallengeID:       challenge.PreviousChallengeID,
		PreviousChallengeTitle:    prevTitle,
		PreviousChallengeCategory: prevCategory,
		IsActive:                  challenge.IsActive,
		IsLocked:                  true,
	}
}

func newTeamResponse(team *models.Team) teamResponse {
	return teamResponse{
		ID:         team.ID,
		Name:       team.Name,
		DivisionID: team.DivisionID,
		CreatedAt:  team.CreatedAt,
	}
}
