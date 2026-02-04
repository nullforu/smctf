package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

const (
	redisRefreshPrefix = "refresh:"
)

type AuthService struct {
	cfg                 config.Config
	db                  *bun.DB
	userRepo            *repo.UserRepo
	registrationKeyRepo *repo.RegistrationKeyRepo
	teamRepo            *repo.TeamRepo
	redis               *redis.Client
}

func NewAuthService(cfg config.Config, db *bun.DB, userRepo *repo.UserRepo, registrationKeyRepo *repo.RegistrationKeyRepo, teamRepo *repo.TeamRepo, redis *redis.Client) *AuthService {
	return &AuthService{cfg: cfg, db: db, userRepo: userRepo, registrationKeyRepo: registrationKeyRepo, teamRepo: teamRepo, redis: redis}
}

func (s *AuthService) Register(ctx context.Context, email, username, password, registrationKey, registrationIP string) (*models.User, error) {
	email = normalizeEmail(email)
	username = normalizeTrim(username)
	registrationKey = normalizeTrim(registrationKey)
	registrationIP = normalizeTrim(registrationIP)
	validator := newFieldValidator()
	validator.Required("email", email)
	validator.Required("username", username)
	validator.Required("password", password)
	validator.Required("registration_key", registrationKey)
	validator.Email("email", email)

	if registrationKey != "" && !isSixDigitCode(registrationKey) {
		validator.fields = append(validator.fields, FieldError{Field: "registration_key", Reason: "invalid"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	_, err := s.userRepo.GetByEmailOrUsername(ctx, email, username)
	switch {
	case err == nil:
		return nil, ErrUserExists
	case !errors.Is(err, repo.ErrNotFound):
		return nil, fmt.Errorf("auth.Register lookup: %w", err)
	}

	hash, err := auth.HashPassword(password, s.cfg.PasswordBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		key, err := s.registrationKeyRepo.GetByCodeForUpdate(ctx, tx, registrationKey)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return NewValidationError(FieldError{Field: "registration_key", Reason: "invalid"})
			}

			return fmt.Errorf("auth.Register key lookup: %w", err)
		}

		if key.UsedBy != nil {
			return NewValidationError(FieldError{Field: "registration_key", Reason: "used"})
		}

		user.TeamID = key.TeamID

		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			if db.IsUniqueViolation(err) {
				return ErrUserExists
			}

			return fmt.Errorf("auth.Register create: %w", err)
		}

		var usedByIP *string
		if registrationIP != "" {
			usedByIP = &registrationIP
		}

		usedAt := time.Now().UTC()
		if _, err := tx.NewUpdate().
			Model(key).
			Set("used_by = ?", user.ID).
			Set("used_by_ip = ?", usedByIP).
			Set("used_at = ?", usedAt).
			Where("id = ?", key.ID).
			Exec(ctx); err != nil {
			return fmt.Errorf("auth.Register use key: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) CreateRegistrationKeys(ctx context.Context, adminID int64, count int, teamID int64) ([]models.RegistrationKey, error) {
	validator := newFieldValidator()
	if count < 1 {
		validator.fields = append(validator.fields, FieldError{Field: "count", Reason: "must be >= 1"})
	}

	validator.PositiveID("team_id", teamID)

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.teamRepo.GetByID(ctx, teamID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewValidationError(FieldError{Field: "team_id", Reason: "invalid"})
		}

		return nil, fmt.Errorf("auth.CreateRegistrationKeys team lookup: %w", err)
	}

	created := make([]models.RegistrationKey, 0, count)
	seen := make(map[string]struct{}, count)

	for len(created) < count {
		code, err := generateRegistrationCode()
		if err != nil {
			return nil, fmt.Errorf("auth.CreateRegistrationKeys generate: %w", err)
		}
		if _, ok := seen[code]; ok {
			continue
		}

		key := models.RegistrationKey{
			Code:      code,
			CreatedBy: adminID,
			TeamID:    teamID,
			CreatedAt: time.Now().UTC(),
		}

		if _, err := s.db.NewInsert().Model(&key).Exec(ctx); err != nil {
			if db.IsUniqueViolation(err) {
				continue
			}
			return nil, fmt.Errorf("auth.CreateRegistrationKeys create: %w", err)
		}

		seen[code] = struct{}{}
		created = append(created, key)
	}

	return created, nil
}

func (s *AuthService) ListRegistrationKeys(ctx context.Context) ([]models.RegistrationKeySummary, error) {
	rows, err := s.registrationKeyRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.ListRegistrationKeys: %w", err)
	}

	return rows, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *models.User, error) {
	email = normalizeEmail(email)
	user, err := s.userRepo.GetByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", "", nil, ErrInvalidCreds
		}

		return "", "", nil, fmt.Errorf("auth.Login lookup: %w", err)
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return "", "", nil, ErrInvalidCreds
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return "", "", nil, fmt.Errorf("auth.Login issueTokens: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

func refreshKey(jti string) string {
	return redisRefreshPrefix + jti
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	if err := s.assertRefreshValid(ctx, claims.ID, claims.UserID); err != nil {
		return "", "", ErrInvalidCreds
	}

	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return "", "", fmt.Errorf("auth.Refresh revoke: %w", err)
	}

	user := &models.User{ID: claims.UserID, Role: claims.Role}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return err
	}

	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("auth.Logout revoke: %w", err)
	}

	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (string, string, error) {
	jti := uuid.NewString()
	accessToken, err := auth.GenerateAccessToken(s.cfg.JWT, user.ID, user.Role)

	if err != nil {
		return "", "", fmt.Errorf("auth.issueTokens access: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(s.cfg.JWT, user.ID, user.Role, jti)
	if err != nil {
		return "", "", fmt.Errorf("auth.issueTokens refresh: %w", err)
	}

	if err := s.redis.Set(ctx, refreshKey(jti), strconv.FormatInt(user.ID, 10), s.cfg.JWT.RefreshTTL).Err(); err != nil {
		return "", "", fmt.Errorf("auth.issueTokens store: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) assertRefreshValid(ctx context.Context, jti string, userID int64) error {
	val, err := s.redis.Get(ctx, refreshKey(jti)).Result()
	if err == redis.Nil {
		return ErrInvalidCreds
	}

	if err != nil {
		return fmt.Errorf("auth.assertRefreshValid lookup: %w", err)
	}

	if val == "" {
		return ErrInvalidCreds
	}

	if val != strconv.FormatInt(userID, 10) {
		return ErrInvalidCreds
	}

	return nil
}

func (s *AuthService) parseRefreshToken(refreshToken string) (*auth.Claims, error) {
	claims, err := auth.ParseToken(s.cfg.JWT, refreshToken)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if claims.Type != auth.TokenTypeRefresh || claims.ID == "" {
		return nil, ErrInvalidCreds
	}

	return claims, nil
}
