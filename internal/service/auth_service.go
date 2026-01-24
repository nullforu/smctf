package service

import (
	"context"
	"database/sql"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	cfg      config.Config
	userRepo *repo.UserRepo
	redis    *redis.Client
}

func NewAuthService(cfg config.Config, userRepo *repo.UserRepo, redis *redis.Client) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, redis: redis}
}

func (s *AuthService) Register(ctx context.Context, email, username, password string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)
	var fields []FieldError
	if email == "" {
		fields = append(fields, FieldError{Field: "email", Reason: "required"})
	}
	if username == "" {
		fields = append(fields, FieldError{Field: "username", Reason: "required"})
	}
	if password == "" {
		fields = append(fields, FieldError{Field: "password", Reason: "required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		fields = append(fields, FieldError{Field: "email", Reason: "invalid format"})
	}
	if len(fields) > 0 {
		return nil, NewValidationError(fields...)
	}

	_, err := s.userRepo.GetByEmailOrUsername(ctx, email, username)
	switch {
	case err == nil:
		return nil, ErrUserExists
	case err != sql.ErrNoRows:
		return nil, err
	}

	hash, err := auth.HashPassword(password, s.cfg.PasswordBcryptCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         "user",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil, ErrInvalidCreds
		}
		return "", "", nil, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return "", "", nil, ErrInvalidCreds
	}
	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, user, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := auth.ParseToken(s.cfg.JWT, refreshToken)
	if err != nil {
		return "", "", ErrInvalidCreds
	}
	if claims.Type != auth.TokenTypeRefresh || claims.ID == "" {
		return "", "", ErrInvalidCreds
	}
	if err := s.assertRefreshValid(ctx, claims.ID, claims.UserID); err != nil {
		return "", "", ErrInvalidCreds
	}
	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return "", "", err
	}
	user := &models.User{ID: claims.UserID, Role: claims.Role}
	return s.issueTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := auth.ParseToken(s.cfg.JWT, refreshToken)
	if err != nil {
		return ErrInvalidCreds
	}
	if claims.Type != auth.TokenTypeRefresh || claims.ID == "" {
		return ErrInvalidCreds
	}
	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return err
	}
	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (string, string, error) {
	jti := uuid.NewString()
	accessToken, err := auth.GenerateAccessToken(s.cfg.JWT, user.ID, user.Role)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := auth.GenerateRefreshToken(s.cfg.JWT, user.ID, user.Role, jti)
	if err != nil {
		return "", "", err
	}
	if err := s.redis.Set(ctx, refreshKey(jti), strconv.FormatInt(user.ID, 10), s.cfg.JWT.RefreshTTL).Err(); err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (s *AuthService) assertRefreshValid(ctx context.Context, jti string, userID int64) error {
	val, err := s.redis.Get(ctx, refreshKey(jti)).Result()
	if err == redis.Nil {
		return ErrInvalidCreds
	}
	if err != nil {
		return err
	}
	if val == "" {
		return ErrInvalidCreds
	}
	if val != strconv.FormatInt(userID, 10) {
		return ErrInvalidCreds
	}
	return nil
}

func refreshKey(jti string) string {
	return "refresh:" + jti
}
