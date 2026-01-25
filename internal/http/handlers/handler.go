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

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	accessToken, refreshToken, user, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
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

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	accessToken, refreshToken, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Me(c *gin.Context) {
	userID := middleware.UserID(c)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *Handler) MeSolved(c *gin.Context) {
	userID := middleware.UserID(c)
	rows, err := h.ctf.SolvedChallenges(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) ListChallenges(c *gin.Context) {
	chs, err := h.ctf.ListChallenges(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	resp := make([]gin.H, 0, len(chs))
	for _, ch := range chs {
		resp = append(resp, gin.H{
			"id":          ch.ID,
			"title":       ch.Title,
			"description": ch.Description,
			"points":      ch.Points,
			"is_active":   ch.IsActive,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SubmitFlag(c *gin.Context) {
	challengeID, ok := parseIDParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "id", Reason: "invalid"}},
		})
		return
	}
	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	correct, err := h.ctf.SubmitFlag(c.Request.Context(), middleware.UserID(c), challengeID, req.Flag)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"correct": correct,
	})
}

func (h *Handler) CreateChallenge(c *gin.Context) {
	var req createChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	ch, err := h.ctf.CreateChallenge(c.Request.Context(), req.Title, req.Description, req.Points, req.Flag, active)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":          ch.ID,
		"title":       ch.Title,
		"description": ch.Description,
		"points":      ch.Points,
		"is_active":   ch.IsActive,
	})
}

func (h *Handler) Scoreboard(c *gin.Context) {
	limit := parseLimitQuery(c, 50, 200)
	rows, err := h.users.Scoreboard(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) ScoreboardTimeline(c *gin.Context) {
	limit := parseLimitQuery(c, 50, 200)
	intervalMinutes, err := parseIntervalQuery(c, 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   service.ErrInvalidInput.Error(),
			Details: []service.FieldError{{Field: "interval", Reason: "invalid"}},
		})
		return
	}

	users, err := h.users.Scoreboard(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	userIDs, usernames := indexUsers(users)

	rows, err := h.users.ScoreboardTimeline(c.Request.Context(), userIDs, time.Duration(intervalMinutes)*time.Minute)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ScoreTimelineResponse{
		IntervalMinutes: intervalMinutes,
		Users:           users,
		Buckets:         buildScoreTimelineBuckets(rows, userIDs, usernames),
	})
}
