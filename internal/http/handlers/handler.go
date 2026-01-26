package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"smctf/internal/config"
	"smctf/internal/http/middleware"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	cfg   config.Config
	auth  *service.AuthService
	ctf   *service.CTFService
	users *repo.UserRepo
	redis *redis.Client
}

func New(cfg config.Config, auth *service.AuthService, ctf *service.CTFService, users *repo.UserRepo, redis *redis.Client) *Handler {
	return &Handler{cfg: cfg, auth: auth, ctf: ctf, users: users, redis: redis}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type createChallengeRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Points      int    `json:"points" binding:"required"`
	Flag        string `json:"flag" binding:"required"`
	IsActive    *bool  `json:"is_active"`
}

type submitRequest struct {
	Flag string `json:"flag" binding:"required"`
}

func (h *Handler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	user, err := h.auth.Register(ctx.Request.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
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
	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"role":     user.Role,
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
	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
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
	ctx.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *Handler) MeSolved(ctx *gin.Context) {
	userID := middleware.UserID(ctx)
	rows, err := h.ctf.SolvedChallenges(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) ListChallenges(ctx *gin.Context) {
	challenges, err := h.ctf.ListChallenges(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	resp := make([]gin.H, 0, len(challenges))
	for _, challenge := range challenges {
		resp = append(resp, gin.H{
			"id":          challenge.ID,
			"title":       challenge.Title,
			"description": challenge.Description,
			"points":      challenge.Points,
			"is_active":   challenge.IsActive,
		})
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) SubmitFlag(ctx *gin.Context) {
	challengeID, ok := parseIDParam(ctx, "id")
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "id", Reason: "invalid"}},
		})
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
		go func() {
			bgCtx := context.Background()
			keys, err := h.redis.Keys(bgCtx, "timeline:*").Result()
			if err == nil {
				if len(keys) > 0 {
					_ = h.redis.Del(bgCtx, keys...).Err()
				}
			}
		}()
	}

	ctx.JSON(http.StatusOK, gin.H{
		"correct": correct,
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
	challenge, err := h.ctf.CreateChallenge(ctx.Request.Context(), req.Title, req.Description, req.Points, req.Flag, active)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"id":          challenge.ID,
		"title":       challenge.Title,
		"description": challenge.Description,
		"points":      challenge.Points,
		"is_active":   challenge.IsActive,
	})
}

func (h *Handler) Leaderboard(ctx *gin.Context) {
	rows, err := h.users.Leaderboard(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) Timeline(ctx *gin.Context) {
	windowMinutes, err := parseWindowQuery(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "window", Reason: "invalid"}},
		})
		return
	}

	cacheKey := fmt.Sprintf("timeline:%d", windowMinutes)

	cached, err := h.redis.Get(ctx.Request.Context(), cacheKey).Result()
	if err == nil {
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	var windowStart *time.Time
	if windowMinutes > 0 {
		start := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
		windowStart = &start
	}

	raw, err := h.users.TimelineSubmissions(ctx.Request.Context(), windowStart)
	if err != nil {
		writeError(ctx, err)
		return
	}

	rawSubs := make([]rawSubmission, len(raw))
	for i, r := range raw {
		rawSubs[i] = rawSubmission{
			SubmittedAt: r.SubmittedAt,
			UserID:      r.UserID,
			Username:    r.Username,
			Points:      r.Points,
		}
	}

	submissions := groupSubmissions(rawSubs)
	response := gin.H{
		"submissions": submissions,
	}

	responseJSON, err := json.Marshal(response)
	if err == nil {
		_ = h.redis.Set(ctx.Request.Context(), cacheKey, responseJSON, h.cfg.Cache.TimelineTTL).Err()
	}

	ctx.JSON(http.StatusOK, response)
}
