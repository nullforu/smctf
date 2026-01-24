package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
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
	"github.com/go-playground/validator/v10"
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

type errorResponse struct {
	Error   string               `json:"error"`
	Details []service.FieldError `json:"details,omitempty"`
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	msg := "internal error"
	var details []service.FieldError

	var ve *service.ValidationError
	if errors.As(err, &ve) {
		status = http.StatusBadRequest
		msg = ve.Error()
		details = ve.Fields
		c.JSON(status, errorResponse{Error: msg, Details: details})
		return
	}

	switch err {
	case service.ErrInvalidInput:
		status = http.StatusBadRequest
		msg = err.Error()
		details = []service.FieldError{{Field: "request", Reason: "invalid"}}
	case service.ErrInvalidCreds:
		status = http.StatusUnauthorized
		msg = err.Error()
	case service.ErrUserExists:
		status = http.StatusConflict
		msg = err.Error()
	case service.ErrChallengeNotFound:
		status = http.StatusNotFound
		msg = err.Error()
	case service.ErrAlreadySolved:
		status = http.StatusConflict
		msg = err.Error()
	case service.ErrRateLimited:
		status = http.StatusTooManyRequests
		msg = err.Error()
	case sql.ErrNoRows:
		status = http.StatusNotFound
		msg = "not found"
	default:
		msg = "internal error"
	}

	c.JSON(status, errorResponse{Error: msg, Details: details})
}

func writeBindError(c *gin.Context, err error) {
	fields := bindErrorDetails(err)
	if len(fields) == 0 {
		fields = []service.FieldError{{Field: "body", Reason: "invalid"}}
	}
	c.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: fields})
}

func bindErrorDetails(err error) []service.FieldError {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make([]service.FieldError, 0, len(verrs))
		for _, fe := range verrs {
			field := strings.ToLower(fe.Field())
			fields = append(fields, service.FieldError{Field: field, Reason: fe.Tag()})
		}
		return fields
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		field := strings.ToLower(ute.Field)
		if field == "" {
			field = "body"
		}
		return []service.FieldError{{Field: field, Reason: "invalid type"}}
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return []service.FieldError{{Field: "body", Reason: "invalid json"}}
	}
	if errors.Is(err, io.EOF) {
		return []service.FieldError{{Field: "body", Reason: "empty"}}
	}
	return nil
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
	challengeID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || challengeID <= 0 {
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
	limit := 50
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	rows, err := h.users.Scoreboard(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	resp := make([]models.ScoreEntry, 0, len(rows))
	resp = append(resp, rows...)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ScoreboardTimeline(c *gin.Context) {
	limit := 50
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	intervalMinutes := 10
	if v := strings.TrimSpace(c.Query("interval")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalMinutes = n
		} else {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error:   service.ErrInvalidInput.Error(),
				Details: []service.FieldError{{Field: "interval", Reason: "invalid"}},
			})
			return
		}
	}

	users, err := h.users.Scoreboard(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	userIDs := make([]int64, 0, len(users))
	usernames := make(map[int64]string, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.UserID)
		usernames[u.UserID] = u.Username
	}

	rows, err := h.users.ScoreboardTimeline(c.Request.Context(), userIDs, time.Duration(intervalMinutes)*time.Minute)
	if err != nil {
		writeError(c, err)
		return
	}

	bucketMap := make(map[time.Time]map[int64]int)
	for _, row := range rows {
		if _, ok := bucketMap[row.Bucket]; !ok {
			bucketMap[row.Bucket] = make(map[int64]int)
		}
		bucketMap[row.Bucket][row.UserID] += row.Score
	}

	buckets := make([]time.Time, 0, len(bucketMap))
	for b := range bucketMap {
		buckets = append(buckets, b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Before(buckets[j]) })

	cumulative := make(map[int64]int, len(userIDs))
	respBuckets := make([]models.ScoreTimelineBucket, 0, len(buckets))
	for _, bucket := range buckets {
		scores := make([]models.ScoreEntry, 0, len(userIDs))
		for _, id := range userIDs {
			cumulative[id] += bucketMap[bucket][id]
			scores = append(scores, models.ScoreEntry{
				UserID:   id,
				Username: usernames[id],
				Score:    cumulative[id],
			})
		}
		respBuckets = append(respBuckets, models.ScoreTimelineBucket{
			Bucket: bucket,
			Scores: scores,
		})
	}

	c.JSON(http.StatusOK, models.ScoreTimelineResponse{
		IntervalMinutes: intervalMinutes,
		Users:           users,
		Buckets:         respBuckets,
	})
}
