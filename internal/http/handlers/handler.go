package handlers

import (
	"net/http"
	"time"

	"smctf/internal/config"
	"smctf/internal/http/middleware"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg   config.Config
	auth  *service.AuthService
	ctf   *service.CTFService
	users *repo.UserRepo
}

func New(cfg config.Config, auth *service.AuthService, ctf *service.CTFService, users *repo.UserRepo) *Handler {
	return &Handler{cfg: cfg, auth: auth, ctf: ctf, users: users}
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

func (h *Handler) Scoreboard(ctx *gin.Context) {
	limit := parseLimitQuery(ctx, 50, 200)
	rows, err := h.users.Scoreboard(ctx.Request.Context(), limit)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) ScoreboardTimeline(ctx *gin.Context) {
	limit := parseLimitQuery(ctx, 50, 200)
	intervalMinutes, err := parseIntervalQuery(ctx, 10)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "interval", Reason: "invalid"}},
		})
		return
	}

	users, err := h.users.Scoreboard(ctx.Request.Context(), limit)
	if err != nil {
		writeError(ctx, err)
		return
	}
	userIDs, usernames := indexUsers(users)

	rows, err := h.users.ScoreboardTimeline(ctx.Request.Context(), userIDs, time.Duration(intervalMinutes)*time.Minute)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, models.ScoreTimelineResponse{
		IntervalMinutes: intervalMinutes,
		Users:           users,
		Buckets:         buildScoreTimelineBuckets(rows, userIDs, usernames),
	})
}
