package service

import (
	"context"
	"errors"
	"fmt"
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

type RegistrationKeyExportItem struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	TeamName  string    `json:"team_name"`
	MaxUses   int       `json:"max_uses"`
	CreatedAt time.Time `json:"created_at"`
}

type RegistrationKeyExportBundle struct {
	Version      int                         `json:"version"`
	ExportedAt   time.Time                   `json:"exported_at"`
	RequestedIDs []int64                     `json:"requested_ids,omitempty"`
	Keys         []RegistrationKeyExportItem `json:"registration_keys"`
}

func NewAuthService(cfg config.Config, db *bun.DB, userRepo *repo.UserRepo, registrationKeyRepo *repo.RegistrationKeyRepo, teamRepo *repo.TeamRepo, redis *redis.Client) *AuthService {
	return &AuthService{cfg: cfg, db: db, userRepo: userRepo, registrationKeyRepo: registrationKeyRepo, teamRepo: teamRepo, redis: redis}
}

func (s *AuthService) Register(ctx context.Context, email, username, password, registrationKey, registrationIP string) (*models.User, error) {
	email = normalizeEmail(email)
	username = normalizeTrim(username)
	registrationKey = strings.ToUpper(normalizeTrim(registrationKey))
	registrationIP = normalizeTrim(registrationIP)
	validator := newFieldValidator()
	validator.Required("email", email)
	validator.Required("username", username)
	validator.MaxLen("username", username, nameMaxLen)
	validator.Required("password", password)
	validator.Required("registration_key", registrationKey)
	validator.Email("email", email)
	validator.MaxBytes("password", password, bcryptInputMaxBytes)

	if registrationKey != "" && !isRegistrationCode(registrationKey) {
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

	hash, err := auth.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         models.UserRole,
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

		if key.UsedCount >= key.MaxUses {
			return NewValidationError(FieldError{Field: "registration_key", Reason: "used"})
		}

		user.TeamID = key.TeamID

		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			if db.IsUniqueViolation(err) {
				return ErrUserExists
			}

			return fmt.Errorf("auth.Register create: %w", err)
		}

		usedAt := time.Now().UTC()
		use := models.RegistrationKeyUse{
			RegistrationKeyID: key.ID,
			UsedBy:            user.ID,
			UsedByIP:          registrationIP,
			UsedAt:            usedAt,
		}

		if _, err := tx.NewInsert().Model(&use).Exec(ctx); err != nil {
			return fmt.Errorf("auth.Register use key: %w", err)
		}

		if _, err := tx.NewUpdate().
			Model(key).
			Set("used_count = used_count + 1").
			Where("id = ?", key.ID).
			Exec(ctx); err != nil {
			return fmt.Errorf("auth.Register increment key usage: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) CreateRegistrationKeys(ctx context.Context, adminID int64, count int, teamID int64, maxUses int) ([]models.RegistrationKey, error) {
	validator := newFieldValidator()
	if count < 1 {
		validator.fields = append(validator.fields, FieldError{Field: "count", Reason: "must be >= 1"})
	}

	validator.PositiveID("team_id", teamID)
	if maxUses < 1 {
		validator.fields = append(validator.fields, FieldError{Field: "max_uses", Reason: "must be >= 1"})
	}

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
			MaxUses:   maxUses,
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

func (s *AuthService) ExportRegistrationKeys(ctx context.Context, ids []int64) (*RegistrationKeyExportBundle, error) {
	validator := newFieldValidator()
	seen := make(map[int64]struct{}, len(ids))
	normalizedIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		validator.PositiveID("ids", id)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalizedIDs = append(normalizedIDs, id)
	}
	if err := validator.Error(); err != nil {
		return nil, err
	}

	rows, err := s.registrationKeyRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.ExportRegistrationKeys: %w", err)
	}

	selected := rows
	if len(normalizedIDs) > 0 {
		byID := make(map[int64]models.RegistrationKeySummary, len(rows))
		for _, row := range rows {
			byID[row.ID] = row
		}

		selected = make([]models.RegistrationKeySummary, 0, len(normalizedIDs))
		for _, id := range normalizedIDs {
			row, ok := byID[id]
			if !ok {
				return nil, NewValidationError(FieldError{Field: "ids", Reason: "contains unknown registration key id"})
			}
			selected = append(selected, row)
		}
	}

	items := make([]RegistrationKeyExportItem, 0, len(selected))
	for _, row := range selected {
		items = append(items, RegistrationKeyExportItem{
			ID:        row.ID,
			Code:      row.Code,
			TeamName:  row.TeamName,
			MaxUses:   row.MaxUses,
			CreatedAt: row.CreatedAt,
		})
	}

	return &RegistrationKeyExportBundle{
		Version:      1,
		ExportedAt:   time.Now().UTC(),
		RequestedIDs: normalizedIDs,
		Keys:         items,
	}, nil
}

func (s *AuthService) ImportRegistrationKeys(ctx context.Context, adminID int64, bundle RegistrationKeyExportBundle) ([]models.RegistrationKeySummary, error) {
	if bundle.Version != 1 {
		return nil, NewValidationError(FieldError{Field: "version", Reason: "unsupported"})
	}
	if len(bundle.Keys) == 0 {
		return nil, NewValidationError(FieldError{Field: "registration_keys", Reason: "required"})
	}

	validator := newFieldValidator()
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth.ImportRegistrationKeys admin lookup: %w", err)
	}
	seenIDs := make(map[int64]struct{}, len(bundle.Keys))
	seenCodes := make(map[string]struct{}, len(bundle.Keys))
	importItems := make([]models.RegistrationKey, 0, len(bundle.Keys))
	teamNames := make([]string, 0, len(bundle.Keys))

	for i, item := range bundle.Keys {
		fieldPrefix := fmt.Sprintf("registration_keys[%d]", i)
		code := strings.ToUpper(normalizeTrim(item.Code))
		teamName := normalizeTrim(item.TeamName)

		validator.PositiveID(fieldPrefix+".id", item.ID)
		validator.Required(fieldPrefix+".code", code)
		validator.Required(fieldPrefix+".team_name", teamName)
		if code != "" && !isRegistrationCode(code) {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".code", Reason: "invalid"})
		}
		if item.MaxUses < 1 {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".max_uses", Reason: "must be >= 1"})
		}

		if _, ok := seenIDs[item.ID]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".id", Reason: "duplicate"})
		}
		seenIDs[item.ID] = struct{}{}

		if _, ok := seenCodes[code]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".code", Reason: "duplicate"})
		}
		seenCodes[code] = struct{}{}

		if _, err := s.registrationKeyRepo.GetByCode(ctx, code); err == nil {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".code", Reason: "duplicate"})
		} else if !errors.Is(err, repo.ErrNotFound) {
			return nil, fmt.Errorf("auth.ImportRegistrationKeys lookup key: %w", err)
		}

		team, err := s.teamRepo.GetByName(ctx, teamName)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".team_name", Reason: "not found"})
				continue
			}
			return nil, fmt.Errorf("auth.ImportRegistrationKeys lookup team: %w", err)
		}

		importItems = append(importItems, models.RegistrationKey{
			Code:      code,
			CreatedBy: adminID,
			TeamID:    team.ID,
			MaxUses:   item.MaxUses,
			CreatedAt: item.CreatedAt.UTC(),
		})
		teamNames = append(teamNames, teamName)
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	imported, err := s.registrationKeyRepo.ImportRegistrationKeys(ctx, importItems)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "registration_keys", Reason: "duplicate"})
		}
		return nil, fmt.Errorf("auth.ImportRegistrationKeys: %w", err)
	}

	rows := make([]models.RegistrationKeySummary, 0, len(imported))
	for i, key := range imported {
		rows = append(rows, models.RegistrationKeySummary{
			ID:                key.ID,
			Code:              key.Code,
			CreatedBy:         key.CreatedBy,
			CreatedByUsername: admin.Username,
			TeamID:            key.TeamID,
			TeamName:          teamNames[i],
			MaxUses:           key.MaxUses,
			UsedCount:         key.UsedCount,
			CreatedAt:         key.CreatedAt,
		})
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

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", "", ErrInvalidCreds
		}

		return "", "", fmt.Errorf("auth.Refresh lookup: %w", err)
	}

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
