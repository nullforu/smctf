package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"smctf/internal/config"
	"smctf/internal/http/middleware"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	cfg    config.Config
	auth   *service.AuthService
	ctf    *service.CTFService
	app    *service.AppConfigService
	users  *repo.UserRepo
	score  *repo.ScoreboardRepo
	teams  *service.TeamService
	stacks *service.StackService
	redis  *redis.Client
}

func New(cfg config.Config, auth *service.AuthService, ctf *service.CTFService, app *service.AppConfigService, users *repo.UserRepo, score *repo.ScoreboardRepo, teams *service.TeamService, stacks *service.StackService, redis *redis.Client) *Handler {
	return &Handler{cfg: cfg, auth: auth, ctf: ctf, app: app, users: users, score: score, teams: teams, stacks: stacks, redis: redis}
}

func windowStartFromMinutes(windowMinutes int) *time.Time {
	if windowMinutes <= 0 {
		return nil
	}
	start := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	return &start
}

func (h *Handler) respondFromCache(ctx *gin.Context, cacheKey string) bool {
	cached, err := h.redis.Get(ctx.Request.Context(), cacheKey).Result()
	if err != nil {
		return false
	}

	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
	return true
}

func (h *Handler) storeCache(ctx *gin.Context, cacheKey string, response any, ttl time.Duration) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return
	}

	_ = h.redis.Set(ctx.Request.Context(), cacheKey, responseJSON, ttl).Err()
}

func (h *Handler) invalidateTimelineCache() {
	go func() {
		bgCtx := context.Background()
		keys, err := h.redis.Keys(bgCtx, "timeline:*").Result()
		if err != nil || len(keys) == 0 {
			return
		}
		_ = h.redis.Del(bgCtx, keys...).Err()
	}()
}

func (h *Handler) invalidateLeaderboardCache() {
	go func() {
		bgCtx := context.Background()
		keys, err := h.redis.Keys(bgCtx, "leaderboard:*").Result()
		if err != nil || len(keys) == 0 {
			return
		}
		_ = h.redis.Del(bgCtx, keys...).Err()
	}()
}

func parseWindowQuery(ctx *gin.Context) (int, error) {
	value := strings.TrimSpace(ctx.Query("window"))
	if value == "" {
		return 0, nil
	}

	window, err := strconv.Atoi(value)
	if err != nil || window <= 0 {
		return 0, errors.New("invalid window")
	}

	return window, nil
}

func parseIDParam(ctx *gin.Context, name string) (int64, bool) {
	value := strings.TrimSpace(ctx.Param(name))
	if value == "" {
		return 0, false
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func parseIDParamOrError(ctx *gin.Context, name string) (int64, bool) {
	id, ok := parseIDParam(ctx, name)
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: name, Reason: "invalid"}},
		})
		return 0, false
	}
	return id, true
}

func (h *Handler) ctfState(ctx *gin.Context) (service.CTFState, bool) {
	state, err := h.app.CTFState(ctx.Request.Context(), time.Now().UTC())
	if err != nil {
		writeError(ctx, err)
		return service.CTFStateActive, false
	}
	return state, true
}

// App Config Handlers

func (h *Handler) GetConfig(ctx *gin.Context) {
	cfg, updatedAt, etag, err := h.app.Get(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	if match := ctx.GetHeader("If-None-Match"); match != "" && etagMatches(match, etag) {
		ctx.Header("ETag", etag)
		ctx.Header("Cache-Control", "no-cache")
		ctx.Status(http.StatusNotModified)
		return
	}

	ctx.Header("ETag", etag)
	if !updatedAt.IsZero() {
		ctx.Header("Last-Modified", updatedAt.UTC().Format(http.TimeFormat))
	}
	ctx.Header("Cache-Control", "no-cache")

	ctx.JSON(http.StatusOK, appConfigResponse{
		Title:             cfg.Title,
		Description:       cfg.Description,
		HeaderTitle:       cfg.HeaderTitle,
		HeaderDescription: cfg.HeaderDescription,
		CTFStartAt:        cfg.CTFStartAt,
		CTFEndAt:          cfg.CTFEndAt,
		UpdatedAt:         updatedAt.UTC(),
	})
}

func etagMatches(ifNoneMatch, etag string) bool {
	needle := normalizeETag(etag)
	for token := range strings.SplitSeq(ifNoneMatch, ",") {
		trimmed := strings.TrimSpace(token)
		if trimmed == "*" {
			return true
		}
		if normalizeETag(trimmed) == needle {
			return true
		}
	}
	return false
}

func normalizeETag(tag string) string {
	tag = strings.TrimSpace(tag)
	if after, ok := strings.CutPrefix(tag, "W/"); ok {
		tag = strings.TrimSpace(after)
	}
	return strings.Trim(tag, "\"")
}

func (h *Handler) AdminUpdateConfig(ctx *gin.Context) {
	var req adminConfigUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	ctfStartAt := optionalStringValue(req.CTFStartAt)
	ctfEndAt := optionalStringValue(req.CTFEndAt)

	cfg, updatedAt, _, err := h.app.Update(ctx.Request.Context(), req.Title, req.Description, req.HeaderTitle, req.HeaderDescription, ctfStartAt, ctfEndAt)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, appConfigResponse{
		Title:             cfg.Title,
		Description:       cfg.Description,
		HeaderTitle:       cfg.HeaderTitle,
		HeaderDescription: cfg.HeaderDescription,
		CTFStartAt:        cfg.CTFStartAt,
		CTFEndAt:          cfg.CTFEndAt,
		UpdatedAt:         updatedAt.UTC(),
	})
}

func optionalStringValue(value optionalString) *string {
	if !value.Set {
		return nil
	}
	if value.Value == nil {
		empty := ""
		return &empty
	}
	return value.Value
}

// Auth Handlers

func (h *Handler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	ip := ctx.ClientIP()

	user, err := h.auth.Register(ctx.Request.Context(), req.Email, req.Username, req.Password, req.RegistrationKey, ip)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, registerResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	})
}

func (h *Handler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	accessToken, refreshToken, user, err := h.auth.Login(ctx.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: loginUserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

func (h *Handler) Refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	accessToken, refreshToken, err := h.auth.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, refreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *Handler) Logout(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	if err := h.auth.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Me(ctx *gin.Context) {
	userID := middleware.UserID(ctx)
	user, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newUserMeResponse(user))
}

func (h *Handler) UpdateMe(ctx *gin.Context) {
	var req meUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	userID := middleware.UserID(ctx)

	user, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	if req.Username != nil {
		user.Username = *req.Username
	}

	if err := h.users.Update(ctx.Request.Context(), user); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserMeResponse(user))
}

// Challenge Handlers

func (h *Handler) ListChallenges(ctx *gin.Context) {
	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state == service.CTFStateNotStarted {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challenges, err := h.ctf.ListChallenges(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	resp := make([]challengeResponse, 0, len(challenges))
	for _, challenge := range challenges {
		ch := challenge
		resp = append(resp, newChallengeResponse(&ch))
	}
	ctx.JSON(http.StatusOK, challengesListResponse{CTFState: string(state), Challenges: resp})
}

func (h *Handler) SubmitFlag(ctx *gin.Context) {
	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state != service.CTFStateActive {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	var req submitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	correct, err := h.ctf.SubmitFlag(ctx.Request.Context(), middleware.UserID(ctx), challengeID, req.Flag)
	if err != nil {
		writeError(ctx, err)
		return
	}

	if correct {
		h.invalidateTimelineCache()
		h.invalidateLeaderboardCache()
		if h.stacks != nil {
			_ = h.stacks.DeleteStackByUserAndChallenge(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"correct":   correct,
		"ctf_state": string(state),
	})
}

func (h *Handler) CreateStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}

	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state != service.CTFStateActive {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	stackModel, err := h.stacks.GetOrCreateStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newStackResponse(stackModel, string(state)))
}

func (h *Handler) GetStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}

	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state == service.CTFStateNotStarted {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	stackModel, err := h.stacks.GetStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newStackResponse(stackModel, string(state)))
}

func (h *Handler) DeleteStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}

	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state == service.CTFStateNotStarted {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	if err := h.stacks.DeleteStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "ok", "ctf_state": string(state)})
}

func (h *Handler) ListStacks(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}

	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state == service.CTFStateNotStarted {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	stacks, err := h.stacks.ListUserStacks(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]stackResponse, 0, len(stacks))
	for i := range stacks {
		stackModel := stacks[i]
		resp = append(resp, newStackResponse(&stackModel, string(state)))
	}

	ctx.JSON(http.StatusOK, stacksListResponse{CTFState: string(state), Stacks: resp})
}

func (h *Handler) CreateChallenge(ctx *gin.Context) {
	var req createChallengeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	minimumPoints := req.Points
	if req.MinimumPoints != nil {
		minimumPoints = *req.MinimumPoints
	}

	stackEnabled := false
	if req.StackEnabled != nil {
		stackEnabled = *req.StackEnabled
	}

	stackTargetPort := 0
	if req.StackTargetPort != nil {
		stackTargetPort = *req.StackTargetPort
	}

	challenge, err := h.ctf.CreateChallenge(ctx.Request.Context(), req.Title, req.Description, req.Category, req.Points, minimumPoints, req.Flag, active, stackEnabled, stackTargetPort, req.StackPodSpec)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.invalidateLeaderboardCache()
	ctx.JSON(http.StatusCreated, newChallengeResponse(challenge))
}

func (h *Handler) UpdateChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req updateChallengeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	challenge, err := h.ctf.UpdateChallenge(ctx.Request.Context(), challengeID, req.Title, req.Description, req.Category, req.Points, req.MinimumPoints, req.Flag, req.IsActive, req.StackEnabled, req.StackTargetPort, req.StackPodSpec)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.invalidateLeaderboardCache()
	ctx.JSON(http.StatusOK, newChallengeResponse(challenge))
}

func (h *Handler) AdminGetChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	challenge, err := h.ctf.GetChallengeByID(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := adminChallengeResponse{
		challengeResponse: newChallengeResponse(challenge),
		StackPodSpec:      challenge.StackPodSpec,
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	if err := h.ctf.DeleteChallenge(ctx.Request.Context(), challengeID); err != nil {
		writeError(ctx, err)
		return
	}

	h.invalidateLeaderboardCache()
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) RequestChallengeFileUpload(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req challengeFileUploadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	challenge, upload, err := h.ctf.RequestChallengeFileUpload(ctx.Request.Context(), challengeID, req.Filename)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, challengeFileUploadResponse{
		Challenge: newChallengeResponse(challenge),
		Upload: presignedPostResponse{
			URL:       upload.URL,
			Fields:    upload.Fields,
			ExpiresAt: upload.ExpiresAt,
		},
	})
}

func (h *Handler) RequestChallengeFileDownload(ctx *gin.Context) {
	state, ok := h.ctfState(ctx)
	if !ok {
		return
	}

	if state == service.CTFStateNotStarted {
		ctx.JSON(http.StatusOK, ctfStateResponse{CTFState: string(state)})
		return
	}

	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	download, err := h.ctf.RequestChallengeFileDownload(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, presignedURLResponse{
		URL:       download.URL,
		ExpiresAt: download.ExpiresAt,
		CTFState:  string(state),
	})
}

func (h *Handler) DeleteChallengeFile(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	challenge, err := h.ctf.DeleteChallengeFile(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newChallengeResponse(challenge))
}

// Registration Key Handlers

func (h *Handler) CreateRegistrationKeys(ctx *gin.Context) {
	var req createRegistrationKeysRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	count := 0
	if req.Count != nil {
		count = *req.Count
	}

	if req.TeamID == nil {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "team_id", Reason: "required"}))
		return
	}

	teamID := *req.TeamID
	adminID := middleware.UserID(ctx)
	admin, err := h.users.GetByID(ctx.Request.Context(), adminID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	keys, err := h.auth.CreateRegistrationKeys(ctx.Request.Context(), adminID, count, teamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	team, err := h.teams.GetTeam(ctx.Request.Context(), teamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]models.RegistrationKeySummary, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, models.RegistrationKeySummary{
			ID:                key.ID,
			Code:              key.Code,
			CreatedBy:         key.CreatedBy,
			CreatedByUsername: admin.Username,
			TeamID:            key.TeamID,
			TeamName:          team.Name,
			UsedBy:            key.UsedBy,
			UsedByUsername:    nil,
			UsedByIP:          nil,
			CreatedAt:         key.CreatedAt,
			UsedAt:            key.UsedAt,
		})
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListRegistrationKeys(ctx *gin.Context) {
	rows, err := h.auth.ListRegistrationKeys(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}

// Scoreboard Handlers

func aggregateUserTimeline(raw []models.UserTimelineRow) []models.TimelineSubmission {
	if len(raw) == 0 {
		return []models.TimelineSubmission{}
	}

	type teamKey struct {
		userID int64
		bucket time.Time
	}

	teams := make(map[teamKey]*models.TimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{userID: sub.UserID, bucket: bucket}

		if team, exists := teams[key]; exists {
			team.Points += sub.Points
			team.ChallengeCount++
		} else {
			teams[key] = &models.TimelineSubmission{
				Timestamp:      bucket,
				UserID:         sub.UserID,
				Username:       sub.Username,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]models.TimelineSubmission, 0, len(teams))
	for _, team := range teams {
		result = append(result, *team)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			return result[i].UserID < result[j].UserID
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

func aggregateTeamTimeline(raw []models.TeamTimelineRow) []models.TeamTimelineSubmission {
	if len(raw) == 0 {
		return []models.TeamTimelineSubmission{}
	}

	type teamKey struct {
		teamID int64
		bucket time.Time
	}

	teams := make(map[teamKey]*models.TeamTimelineSubmission)

	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := teamKey{teamID: sub.TeamID, bucket: bucket}

		if team, exists := teams[key]; exists {
			team.Points += sub.Points
			team.ChallengeCount++
		} else {
			teams[key] = &models.TeamTimelineSubmission{
				Timestamp:      bucket,
				TeamID:         sub.TeamID,
				TeamName:       sub.TeamName,
				Points:         sub.Points,
				ChallengeCount: 1,
			}
		}
	}

	result := make([]models.TeamTimelineSubmission, 0, len(teams))
	for _, team := range teams {
		result = append(result, *team)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			if result[i].TeamName == result[j].TeamName {
				return result[i].TeamID < result[j].TeamID
			}

			return result[i].TeamName < result[j].TeamName
		}

		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

func parseWindowOrError(ctx *gin.Context) (int, bool) {
	windowMinutes, err := parseWindowQuery(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "window", Reason: "invalid"}},
		})
		return 0, false
	}
	return windowMinutes, true
}

func (h *Handler) Leaderboard(ctx *gin.Context) {
	cacheKey := "leaderboard:users"
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	rows, err := h.score.Leaderboard(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.storeCache(ctx, cacheKey, rows, h.cfg.Cache.LeaderboardTTL)
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) TeamLeaderboard(ctx *gin.Context) {
	cacheKey := "leaderboard:teams"
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	rows, err := h.score.TeamLeaderboard(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.storeCache(ctx, cacheKey, rows, h.cfg.Cache.LeaderboardTTL)
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) Timeline(ctx *gin.Context) {
	windowMinutes, ok := parseWindowOrError(ctx)
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("timeline:%d", windowMinutes)

	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	windowStart := windowStartFromMinutes(windowMinutes)

	raw, err := h.score.TimelineSubmissions(ctx.Request.Context(), windowStart)
	if err != nil {
		writeError(ctx, err)
		return
	}

	submissions := aggregateUserTimeline(raw)
	response := timelineResponse{Submissions: submissions}

	h.storeCache(ctx, cacheKey, response, h.cfg.Cache.TimelineTTL)

	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) TeamTimeline(ctx *gin.Context) {
	windowMinutes, ok := parseWindowOrError(ctx)
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("timeline:teams:%d", windowMinutes)

	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	windowStart := windowStartFromMinutes(windowMinutes)

	raw, err := h.score.TimelineTeamSubmissions(ctx.Request.Context(), windowStart)
	if err != nil {
		writeError(ctx, err)
		return
	}

	submissions := aggregateTeamTimeline(raw)
	response := teamTimelineResponse{Submissions: submissions}

	h.storeCache(ctx, cacheKey, response, h.cfg.Cache.TimelineTTL)

	ctx.JSON(http.StatusOK, response)
}

// Team Handlers

func (h *Handler) CreateTeam(ctx *gin.Context) {
	var req createTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	team, err := h.teams.CreateTeam(ctx.Request.Context(), req.Name)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newTeamResponse(team))
}

func (h *Handler) ListTeams(ctx *gin.Context) {
	teams, err := h.teams.ListTeams(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, teams)
}

func (h *Handler) GetTeam(ctx *gin.Context) {
	teamID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	team, err := h.teams.GetTeam(ctx.Request.Context(), teamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, team)
}

func (h *Handler) ListTeamMembers(ctx *gin.Context) {
	teamID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	rows, err := h.teams.ListMembers(ctx.Request.Context(), teamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) ListTeamSolved(ctx *gin.Context) {
	teamID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	rows, err := h.teams.ListSolvedChallenges(ctx.Request.Context(), teamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}

// User Handlers

func (h *Handler) ListUsers(ctx *gin.Context) {
	users, err := h.users.List(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]userDetailResponse, 0, len(users))
	for _, user := range users {
		u := user
		resp = append(resp, newUserDetailResponse(&u))
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) GetUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	user, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserDetailResponse(user))
}

func (h *Handler) GetUserSolved(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	_, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	rows, err := h.ctf.SolvedChallenges(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}
