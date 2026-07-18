package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/http/middleware"
	"smctf/internal/models"
	"smctf/internal/realtime"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	cfg     config.Config
	auth    *service.AuthService
	ctf     *service.CTFService
	app     *service.AppConfigService
	users   *service.UserService
	score   *service.ScoreboardService
	divs    *service.DivisionService
	teams   *service.TeamService
	vms     *service.VMService
	discord *service.DiscordService
	redis   *redis.Client
}

func New(cfg config.Config, auth *service.AuthService, ctf *service.CTFService, app *service.AppConfigService, users *service.UserService, score *service.ScoreboardService, divisions *service.DivisionService, teams *service.TeamService, vms *service.VMService, redis *redis.Client, discord *service.DiscordService) *Handler {
	return &Handler{cfg: cfg, auth: auth, ctf: ctf, app: app, users: users, score: score, divs: divisions, teams: teams, vms: vms, redis: redis, discord: discord}
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

func cacheKeyWithDivision(base string, divisionID *int64) string {
	if divisionID == nil {
		return base
	}

	return fmt.Sprintf("%s:div:%d", base, *divisionID)
}

func (h *Handler) invalidateCacheByPrefix(ctx context.Context, prefix string) {
	iter := h.redis.Scan(ctx, 0, prefix+"*", 200).Iterator()
	for iter.Next(ctx) {
		_ = h.redis.Del(ctx, iter.Val()).Err()
	}
	if err := iter.Err(); err != nil {
		slog.Warn("cache scan failed", slog.String("prefix", prefix), slog.Any("error", err))
	}
}

func (h *Handler) invalidateTimelineCache(ctx context.Context, divisionIDs []int64) {
	if len(divisionIDs) == 0 {
		h.invalidateCacheByPrefix(ctx, "timeline:")
		return
	}

	for _, divisionID := range divisionIDs {
		divID := divisionID
		_ = h.redis.Del(ctx,
			cacheKeyWithDivision("timeline:users", &divID),
			cacheKeyWithDivision("timeline:teams", &divID),
		).Err()
	}
}

func (h *Handler) invalidateLeaderboardCache(ctx context.Context, divisionIDs []int64) {
	if len(divisionIDs) == 0 {
		h.invalidateCacheByPrefix(ctx, "leaderboard:")
		return
	}

	for _, divisionID := range divisionIDs {
		divID := divisionID
		_ = h.redis.Del(ctx,
			cacheKeyWithDivision("leaderboard:users", &divID),
			cacheKeyWithDivision("leaderboard:teams", &divID),
		).Err()
	}
}

func uniqueDivisionIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))

	for _, id := range ids {
		if id <= 0 {
			continue
		}

		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func (h *Handler) publishScoreboardEvent(ctx context.Context, reason string, divisionIDs []int64) {
	divisionIDs = uniqueDivisionIDs(divisionIDs)
	scope := "all"

	if len(divisionIDs) > 0 {
		scope = "division"
	}

	event := realtime.ScoreboardEvent{
		Scope:       scope,
		Reason:      reason,
		TS:          time.Now().UTC(),
		DivisionIDs: divisionIDs,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	_ = h.redis.Publish(ctx, "scoreboard.events", payload).Err()
}

func (h *Handler) notifyScoreboardChanged(ctx context.Context, reason string, divisionIDs ...int64) {
	divisionIDs = uniqueDivisionIDs(divisionIDs)
	h.invalidateLeaderboardCache(ctx, divisionIDs)
	h.invalidateTimelineCache(ctx, divisionIDs)
	h.publishScoreboardEvent(ctx, reason, divisionIDs)
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

func parseOptionalPositiveIDQuery(ctx *gin.Context, name string) (*int64, error) {
	raw := strings.TrimSpace(ctx.Query(name))
	if raw == "" {
		return nil, nil
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, service.NewValidationError(service.FieldError{Field: name, Reason: "invalid"})
	}

	return &id, nil
}

func parseOptionalPositiveIDsQuery(ctx *gin.Context, name string) ([]int64, error) {
	raw := strings.TrimSpace(ctx.Query(name))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			return nil, service.NewValidationError(service.FieldError{Field: name, Reason: "invalid"})
		}

		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, service.NewValidationError(service.FieldError{Field: name, Reason: "invalid"})
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func (h *Handler) resolveDivisionID(ctx *gin.Context, require bool) (*int64, bool) {
	divisionID, err := parseOptionalPositiveIDQuery(ctx, "division_id")
	if err != nil {
		writeError(ctx, err)
		return nil, false
	}

	if divisionID != nil {
		return divisionID, true
	}

	if require {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "division_id", Reason: "required"}))
		return nil, false
	}

	return nil, true
}

func (h *Handler) optionalUserID(ctx *gin.Context) int64 {
	token, err := ctx.Cookie("access_token")
	if err != nil || token == "" {
		return 0
	}

	claims, err := auth.ParseToken(h.cfg.JWT, token)
	if err != nil || claims.Type != auth.TokenTypeAccess {
		return 0
	}

	return claims.UserID
}

func isChallengeLocked(challenge models.Challenge, solved map[int64]struct{}, userID int64) bool {
	if challenge.PreviousChallengeID == nil || *challenge.PreviousChallengeID <= 0 {
		return false
	}

	if userID <= 0 {
		return true
	}

	_, ok := solved[*challenge.PreviousChallengeID]
	return !ok
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
	ctx.Header("Cache-Control", "no-cache")
	if !updatedAt.IsZero() {
		ctx.Header("Last-Modified", updatedAt.UTC().Format(http.TimeFormat))
	}

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

	input := service.AppConfigUpdate{
		Title:             appConfigInputFromOptional(req.Title),
		Description:       appConfigInputFromOptional(req.Description),
		HeaderTitle:       appConfigInputFromOptional(req.HeaderTitle),
		HeaderDescription: appConfigInputFromOptional(req.HeaderDescription),
		CTFStartAt:        appConfigInputFromOptional(req.CTFStartAt),
		CTFEndAt:          appConfigInputFromOptional(req.CTFEndAt),
	}

	cfg, updatedAt, _, err := h.app.Update(ctx.Request.Context(), input)
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

func appConfigInputFromOptional(value optionalString) service.AppConfigUpdateInput {
	if !value.Set {
		return service.AppConfigUpdateInput{}
	}

	if value.Value == nil {
		return service.AppConfigUpdateInput{Set: true, Null: true}
	}

	return service.AppConfigUpdateInput{Set: true, Value: *value.Value}
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

	h.notifyScoreboardChanged(ctx.Request.Context(), "user_registered", user.DivisionID)

	ctx.JSON(http.StatusCreated, registerResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	})
}

func (h *Handler) userVMSummary(ctx context.Context, userID int64) (int, int) {
	if h.vms == nil {
		return 0, 0
	}

	count, limit, err := h.vms.UserVMSummary(ctx, userID)
	if err != nil {
		slog.Warn("vm summary lookup failed", slog.Int64("user_id", userID), slog.Any("error", err))
		return 0, limit
	}

	return count, limit
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
	if err := setAuthCookies(ctx, h.cfg, accessToken, refreshToken); err != nil {
		writeError(ctx, err)
		return
	}

	vmCount, vmLimit := h.userVMSummary(ctx.Request.Context(), user.ID)

	ctx.JSON(http.StatusOK, loginResponse{
		User: newUserMeResponse(user, vmCount, vmLimit),
	})
}

func (h *Handler) Refresh(ctx *gin.Context) {
	refreshTokenValue, ok := currentRefreshToken(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	accessToken, refreshToken, err := h.auth.Refresh(ctx.Request.Context(), refreshTokenValue)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := setAuthCookies(ctx, h.cfg, accessToken, refreshToken); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, refreshResponse{Status: "ok"})
}

func (h *Handler) Logout(ctx *gin.Context) {
	refreshTokenValue, ok := currentRefreshToken(ctx)
	if ok {
		if err := h.auth.Logout(ctx.Request.Context(), refreshTokenValue); err != nil {
			writeError(ctx, err)
			return
		}
	}

	clearAuthCookies(ctx, h.cfg)
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Me(ctx *gin.Context) {
	userID := middleware.UserID(ctx)
	user, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	vmCount, vmLimit := h.userVMSummary(ctx.Request.Context(), userID)

	ctx.JSON(http.StatusOK, newUserMeResponse(user, vmCount, vmLimit))
}

func (h *Handler) UpdateMe(ctx *gin.Context) {
	var req meUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	userID := middleware.UserID(ctx)
	user, err := h.users.UpdateProfile(ctx.Request.Context(), userID, req.Username)
	if err != nil {
		writeError(ctx, err)
		return
	}

	vmCount, vmLimit := h.userVMSummary(ctx.Request.Context(), userID)

	h.notifyScoreboardChanged(ctx.Request.Context(), "user_profile_update", user.DivisionID)

	if h.discord != nil && req.Username != nil {
		h.discord.SyncNickname(ctx.Request.Context(), userID)
	}

	ctx.JSON(http.StatusOK, newUserMeResponse(user, vmCount, vmLimit))
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

	divisionID, ok := h.resolveDivisionID(ctx, true)
	if !ok {
		return
	}

	challenges, err := h.ctf.ListChallenges(ctx.Request.Context(), divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	userID := h.optionalUserID(ctx)
	solved := map[int64]struct{}{}

	hasPrereq := false
	for i := range challenges {
		if challenges[i].PreviousChallengeID != nil && *challenges[i].PreviousChallengeID > 0 {
			hasPrereq = true
			break
		}
	}

	if userID > 0 && hasPrereq {
		solved, err = h.ctf.TeamSolvedChallengeIDs(ctx.Request.Context(), userID)
		if err != nil {
			writeError(ctx, err)
			return
		}
	}

	resp := make([]any, 0, len(challenges))
	byID := make(map[int64]*models.Challenge, len(challenges))
	for i := range challenges {
		byID[challenges[i].ID] = &challenges[i]
	}

	for _, challenge := range challenges {
		ch := challenge
		if isChallengeLocked(ch, solved, userID) {
			var previous *models.Challenge
			if ch.PreviousChallengeID != nil {
				previous = byID[*ch.PreviousChallengeID]
			}

			resp = append(resp, newLockedChallengeResponse(&ch, previous))
			continue
		}

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
		divisionID, err := h.users.GetDivisionID(ctx.Request.Context(), middleware.UserID(ctx))
		if err != nil {
			h.notifyScoreboardChanged(ctx.Request.Context(), "submission_correct")
		} else {
			h.notifyScoreboardChanged(ctx.Request.Context(), "submission_correct", divisionID)
		}

		if h.vms != nil {
			_ = h.vms.DeleteVMByUserAndChallenge(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
		}

		if h.discord != nil {
			uid := middleware.UserID(ctx)
			if ch, cerr := h.ctf.GetChallengeByID(ctx.Request.Context(), challengeID); cerr == nil && ch.Title != "" {
				firstBlood, _ := h.ctf.WasFirstBlood(ctx.Request.Context(), uid, challengeID)
				h.discord.AnnounceSolve(ctx.Request.Context(), uid, ch.Title, firstBlood)
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"correct":   correct,
		"ctf_state": string(state),
	})
}

func (h *Handler) CreateVM(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
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

	vmModel, err := h.vms.GetOrCreateVM(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newVMResponse(vmModel, string(state)))
}

func (h *Handler) GetVM(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
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

	vmModel, err := h.vms.GetVM(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newVMResponse(vmModel, string(state)))
}

func (h *Handler) DeleteVM(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
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

	if err := h.vms.DeleteVM(ctx.Request.Context(), middleware.UserID(ctx), challengeID); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "ok", "ctf_state": string(state)})
}

func (h *Handler) ListVMs(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
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

	vms, err := h.vms.ListUserVMs(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]vmResponse, 0, len(vms))
	for i := range vms {
		vmModel := vms[i]
		resp = append(resp, newVMResponse(&vmModel, string(state)))
	}

	ctx.JSON(http.StatusOK, vmsListResponse{CTFState: string(state), VMs: resp})
}

func (h *Handler) AdminListVMs(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
		return
	}

	vms, err := h.vms.ListAdminVMs(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]adminVMResponse, 0, len(vms))
	for i := range vms {
		resp = append(resp, newAdminVMResponse(vms[i]))
	}

	ctx.JSON(http.StatusOK, adminVMsListResponse{VMs: resp})
}

func (h *Handler) AdminDeleteVM(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
		return
	}

	vmID := strings.TrimSpace(ctx.Param("vm_id"))
	if vmID == "" {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "vm_id", Reason: "required"}))
		return
	}

	if err := h.vms.DeleteVMByVMID(ctx.Request.Context(), vmID); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"deleted": true, "vm_id": vmID})
}

func (h *Handler) AdminGetVM(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
		return
	}

	vmID := strings.TrimSpace(ctx.Param("vm_id"))
	if vmID == "" {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "vm_id", Reason: "required"}))
		return
	}

	vmModel, err := h.vms.GetVMByVMID(ctx.Request.Context(), vmID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newVMResponse(vmModel, ""))
}

func (h *Handler) AdminReport(ctx *gin.Context) {
	if h.vms == nil {
		writeError(ctx, service.ErrVMDisabled)
		return
	}

	challenges, err := h.ctf.ListChallenges(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	divisions, err := h.divs.ListDivisions(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	teams, err := h.teams.ListTeams(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	users, err := h.users.List(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	vms, err := h.vms.ListAllVMs(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	keys, err := h.auth.ListRegistrationKeys(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	submissions, err := h.ctf.ListAllSubmissions(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	appConfigs, err := h.app.GetAllRows(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	leaderboard, err := h.score.Leaderboard(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	teamLeaderboard, err := h.score.TeamLeaderboard(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	reportChallenges := make([]adminReportChallenge, 0, len(challenges))
	for i := range challenges {
		reportChallenges = append(reportChallenges, newAdminReportChallenge(challenges[i]))
	}

	reportUsers := make([]adminReportUser, 0, len(users))
	for i := range users {
		reportUsers = append(reportUsers, newAdminReportUser(users[i]))
	}

	reportSubmissions := make([]adminReportSubmission, 0, len(submissions))
	for i := range submissions {
		reportSubmissions = append(reportSubmissions, newAdminReportSubmission(submissions[i]))
	}

	userTimeline, err := h.score.UserTimeline(ctx.Request.Context(), nil, nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	teamTimelineRows, err := h.score.TeamTimeline(ctx.Request.Context(), nil, nil)
	if err != nil {
		writeError(ctx, err)
		return
	}

	timeline := timelineResponse{Submissions: userTimeline}
	teamTimeline := teamTimelineResponse{Submissions: teamTimelineRows}

	ctx.JSON(http.StatusOK, adminReportResponse{
		Challenges:       reportChallenges,
		Divisions:        divisions,
		Teams:            teams,
		Users:            reportUsers,
		VMs:              vms,
		RegistrationKeys: keys,
		Submissions:      reportSubmissions,
		AppConfig:        appConfigs,
		Timeline:         timeline,
		TeamTimeline:     teamTimeline,
		Leaderboard:      leaderboard,
		TeamLeaderboard:  teamLeaderboard,
	})
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

	vmEnabled := false
	if req.VMEnabled != nil {
		vmEnabled = *req.VMEnabled
	}

	challenge, err := h.ctf.CreateChallenge(ctx.Request.Context(), req.Title, req.Description, req.Category, req.Points, minimumPoints, req.Flag, active, vmEnabled, req.VMSpec, req.PreviousChallengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_created")
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

	title, err := requireNonNullOptionalString("title", req.Title)
	if err != nil {
		writeError(ctx, err)
		return
	}

	description, err := requireNonNullOptionalString("description", req.Description)
	if err != nil {
		writeError(ctx, err)
		return
	}

	category, err := requireNonNullOptionalString("category", req.Category)
	if err != nil {
		writeError(ctx, err)
		return
	}

	flag, err := requireNonNullOptionalString("flag", req.Flag)
	if err != nil {
		writeError(ctx, err)
		return
	}

	vmSpec := optionalStringToPointer(req.VMSpec)

	previousChallengeID := (*int64)(nil)
	previousChallengeSet := req.PreviousChallengeID.Set
	if previousChallengeSet {
		previousChallengeID = req.PreviousChallengeID.Value
	}

	challenge, err := h.ctf.UpdateChallenge(ctx.Request.Context(), challengeID, title, description, category, req.Points, req.MinimumPoints, flag, req.IsActive, req.VMEnabled, vmSpec, previousChallengeID, previousChallengeSet)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_updated")
	ctx.JSON(http.StatusOK, newChallengeResponse(challenge))
}

func requireNonNullOptionalString(field string, value optionalString) (*string, error) {
	if !value.Set {
		return nil, nil
	}

	if value.Value == nil {
		return nil, service.NewValidationError(service.FieldError{Field: field, Reason: "invalid"})
	}

	return value.Value, nil
}

func optionalStringToPointer(value optionalString) *string {
	if !value.Set {
		return nil
	}

	if value.Value == nil {
		empty := ""
		return &empty
	}

	return value.Value
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
		VMSpec:            challenge.VMSpec,
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

	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_deleted")
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ExportChallenges(ctx *gin.Context) {
	ids, err := parseOptionalPositiveIDsQuery(ctx, "ids")
	if err != nil {
		writeError(ctx, err)
		return
	}

	bundle, err := h.ctf.ExportChallenges(ctx.Request.Context(), ids)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, challengeExportResponse{
		Version:      bundle.Version,
		ExportedAt:   bundle.ExportedAt,
		RequestedIDs: bundle.RequestedIDs,
		Challenges:   bundle.Challenges,
	})
}

func (h *Handler) ImportChallenges(ctx *gin.Context) {
	var req challengeImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	imported, err := h.ctf.ImportChallenges(ctx.Request.Context(), service.ChallengeExportBundle(req))
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_imported")

	response := make([]challengeResponse, 0, len(imported))
	for i := range imported {
		response = append(response, newChallengeResponse(&imported[i]))
	}

	ctx.JSON(http.StatusCreated, challengeImportResponse{Imported: response})
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

	download, err := h.ctf.RequestChallengeFileDownload(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
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

	maxUses := 1
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
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

	keys, err := h.auth.CreateRegistrationKeys(ctx.Request.Context(), adminID, count, teamID, maxUses)
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
			MaxUses:           key.MaxUses,
			UsedCount:         key.UsedCount,
			CreatedAt:         key.CreatedAt,
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

func (h *Handler) ExportRegistrationKeys(ctx *gin.Context) {
	ids, err := parseOptionalPositiveIDsQuery(ctx, "ids")
	if err != nil {
		writeError(ctx, err)
		return
	}

	bundle, err := h.auth.ExportRegistrationKeys(ctx.Request.Context(), ids)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, registrationKeyExportResponse{
		Version:      bundle.Version,
		ExportedAt:   bundle.ExportedAt,
		RequestedIDs: bundle.RequestedIDs,
		Keys:         bundle.Keys,
	})
}

func (h *Handler) ImportRegistrationKeys(ctx *gin.Context) {
	var req registrationKeyImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	adminID := middleware.UserID(ctx)
	imported, err := h.auth.ImportRegistrationKeys(ctx.Request.Context(), adminID, service.RegistrationKeyExportBundle(req))
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, registrationKeyImportResponse{Imported: imported})
}

func (h *Handler) AdminMoveUserTeam(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req adminMoveUserTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	beforeDivisionID, err := h.users.GetDivisionID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	updated, err := h.users.MoveUserTeam(ctx.Request.Context(), userID, req.TeamID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	if updated.DivisionID != beforeDivisionID {
		h.notifyScoreboardChanged(ctx.Request.Context(), "user_team_moved", beforeDivisionID, updated.DivisionID)
	} else {
		h.notifyScoreboardChanged(ctx.Request.Context(), "user_team_moved", updated.DivisionID)
	}

	ctx.JSON(http.StatusOK, newAdminUserResponse(updated))
}

func (h *Handler) AdminBlockUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req adminBlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	updated, err := h.users.BlockUser(ctx.Request.Context(), userID, req.Reason)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "user_blocked", updated.DivisionID)

	ctx.JSON(http.StatusOK, newAdminUserResponse(updated))
}

func (h *Handler) AdminUnblockUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	updated, err := h.users.UnblockUser(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "user_unblocked", updated.DivisionID)

	ctx.JSON(http.StatusOK, newAdminUserResponse(updated))
}

// Scoreboard Handlers

func (h *Handler) Leaderboard(ctx *gin.Context) {
	divisionID, ok := h.resolveDivisionID(ctx, true)
	if !ok {
		return
	}

	cacheKey := cacheKeyWithDivision("leaderboard:users", divisionID)
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	rows, err := h.score.Leaderboard(ctx.Request.Context(), divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.storeCache(ctx, cacheKey, rows, h.cfg.Cache.LeaderboardTTL)
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) TeamLeaderboard(ctx *gin.Context) {
	divisionID, ok := h.resolveDivisionID(ctx, true)
	if !ok {
		return
	}

	cacheKey := cacheKeyWithDivision("leaderboard:teams", divisionID)
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	rows, err := h.score.TeamLeaderboard(ctx.Request.Context(), divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.storeCache(ctx, cacheKey, rows, h.cfg.Cache.LeaderboardTTL)
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) Timeline(ctx *gin.Context) {
	divisionID, ok := h.resolveDivisionID(ctx, true)
	if !ok {
		return
	}

	cacheKey := cacheKeyWithDivision("timeline:users", divisionID)
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	submissions, err := h.score.UserTimeline(ctx.Request.Context(), nil, divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	response := timelineResponse{Submissions: submissions}
	h.storeCache(ctx, cacheKey, response, h.cfg.Cache.TimelineTTL)

	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) TeamTimeline(ctx *gin.Context) {
	divisionID, ok := h.resolveDivisionID(ctx, true)
	if !ok {
		return
	}

	cacheKey := cacheKeyWithDivision("timeline:teams", divisionID)
	if h.respondFromCache(ctx, cacheKey) {
		return
	}

	submissions, err := h.score.TeamTimeline(ctx.Request.Context(), nil, divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	response := teamTimelineResponse{Submissions: submissions}

	h.storeCache(ctx, cacheKey, response, h.cfg.Cache.TimelineTTL)

	ctx.JSON(http.StatusOK, response)
}

// Division Handlers

func (h *Handler) ListDivisions(ctx *gin.Context) {
	rows, err := h.divs.ListDivisions(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) CreateDivision(ctx *gin.Context) {
	var req createDivisionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	division, err := h.divs.CreateDivision(ctx.Request.Context(), req.Name, req.DiscordRoleID, req.DiscordAnnounceChannelID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, division)
}

func (h *Handler) UpdateDivision(ctx *gin.Context) {
	id, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req updateDivisionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	division, err := h.divs.UpdateDivision(ctx.Request.Context(), id, req.Name, req.DiscordRoleID, req.DiscordAnnounceChannelID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, division)
}

func (h *Handler) ExportDivisions(ctx *gin.Context) {
	ids, err := parseOptionalPositiveIDsQuery(ctx, "ids")
	if err != nil {
		writeError(ctx, err)
		return
	}

	bundle, err := h.divs.ExportDivisions(ctx.Request.Context(), ids)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, divisionExportResponse{
		Version:      bundle.Version,
		ExportedAt:   bundle.ExportedAt,
		RequestedIDs: bundle.RequestedIDs,
		Divisions:    bundle.Divisions,
	})
}

func (h *Handler) ImportDivisions(ctx *gin.Context) {
	var req divisionImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	imported, err := h.divs.ImportDivisions(ctx.Request.Context(), service.DivisionExportBundle(req))
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, divisionImportResponse{Imported: imported})
}

// Team Handlers

func (h *Handler) CreateTeam(ctx *gin.Context) {
	var req createTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	team, err := h.teams.CreateTeam(ctx.Request.Context(), req.Name, req.DivisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "team_created", team.DivisionID)

	ctx.JSON(http.StatusCreated, newTeamResponse(team))
}

func (h *Handler) ExportTeams(ctx *gin.Context) {
	ids, err := parseOptionalPositiveIDsQuery(ctx, "ids")
	if err != nil {
		writeError(ctx, err)
		return
	}

	bundle, err := h.teams.ExportTeams(ctx.Request.Context(), ids)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, teamExportResponse{
		Version:      bundle.Version,
		ExportedAt:   bundle.ExportedAt,
		RequestedIDs: bundle.RequestedIDs,
		Teams:        bundle.Teams,
	})
}

func (h *Handler) ImportTeams(ctx *gin.Context) {
	var req teamImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	imported, err := h.teams.ImportTeams(ctx.Request.Context(), service.TeamExportBundle(req))
	if err != nil {
		writeError(ctx, err)
		return
	}

	response := make([]teamResponse, 0, len(imported))
	for i := range imported {
		response = append(response, newTeamResponse(&imported[i]))
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "team_imported")
	ctx.JSON(http.StatusCreated, teamImportResponse{Imported: response})
}

func (h *Handler) ListTeams(ctx *gin.Context) {
	divisionID, err := parseOptionalPositiveIDQuery(ctx, "division_id")
	if err != nil {
		writeError(ctx, err)
		return
	}

	teams, err := h.teams.ListTeams(ctx.Request.Context(), divisionID)
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
	divisionID, err := parseOptionalPositiveIDQuery(ctx, "division_id")
	if err != nil {
		writeError(ctx, err)
		return
	}

	users, err := h.users.List(ctx.Request.Context(), divisionID)
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

	divisionID, err := h.users.GetDivisionID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	rows, err := h.ctf.SolvedChallenges(ctx.Request.Context(), userID, &divisionID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rows)
}
