package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"
)

type UserService struct {
	userRepo *repo.UserRepo
	teamRepo *repo.TeamRepo
}

func NewUserService(userRepo *repo.UserRepo, teamRepo *repo.TeamRepo) *UserService {
	return &UserService{userRepo: userRepo, teamRepo: teamRepo}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.GetByID: %w", err)
	}

	return user, nil
}

func (s *UserService) GetDivisionID(ctx context.Context, userID int64) (int64, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	if err := validator.Error(); err != nil {
		return 0, err
	}

	divisionID, err := s.userRepo.GetDivisionID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return 0, ErrNotFound
		}

		return 0, fmt.Errorf("user.GetDivisionID: %w", err)
	}

	return divisionID, nil
}

func (s *UserService) List(ctx context.Context, divisionID *int64) ([]models.User, error) {
	if divisionID != nil {
		validator := newFieldValidator()
		validator.PositiveID("division_id", *divisionID)
		if err := validator.Error(); err != nil {
			return nil, err
		}
	}

	users, err := s.userRepo.List(ctx, divisionID)
	if err != nil {
		return nil, fmt.Errorf("user.List: %w", err)
	}

	return users, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int64, username *string) (*models.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != nil {
		normalizedUsername := normalizeTrim(*username)
		if normalizedUsername == "" {
			return nil, NewValidationError(FieldError{Field: "username", Reason: "required"})
		}

		if utf8.RuneCountInString(normalizedUsername) > nameMaxLen {
			return nil, NewValidationError(FieldError{Field: "username", Reason: fmt.Sprintf("max length is %d characters", nameMaxLen)})
		}

		exists, err := s.userRepo.ExistsByUsername(ctx, normalizedUsername, &userID)
		if err != nil {
			return nil, fmt.Errorf("user.UpdateProfile username exists: %w", err)
		}
		if exists {
			return nil, ErrUserExists
		}

		user.Username = normalizedUsername
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	return user, nil
}

func (s *UserService) MoveUserTeam(ctx context.Context, userID, teamID int64) (*models.User, error) {
	validator := newFieldValidator()
	validator.PositiveID("team_id", teamID)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.teamRepo.GetByID(ctx, teamID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewValidationError(FieldError{Field: "team_id", Reason: "invalid"})
		}

		return nil, fmt.Errorf("user.MoveUserTeam team lookup: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.MoveUserTeam user lookup: %w", err)
	}

	user.TeamID = teamID
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.MoveUserTeam update: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.MoveUserTeam reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) BlockUser(ctx context.Context, userID int64, reason string) (*models.User, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, NewValidationError(FieldError{Field: "reason", Reason: "required"})
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.BlockUser lookup: %w", err)
	}

	if user.Role == models.AdminRole {
		return nil, NewValidationError(FieldError{Field: "user_id", Reason: "admin_blocked"})
	}

	user.Role = models.BlockedRole
	user.BlockedReason = &reason
	blockedAt := time.Now().UTC()
	user.BlockedAt = &blockedAt
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.BlockUser update: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.BlockUser reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) UnblockUser(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.UnblockUser lookup: %w", err)
	}

	if user.Role == models.AdminRole {
		return nil, NewValidationError(FieldError{Field: "user_id", Reason: "admin_blocked"})
	}

	user.Role = models.UserRole
	user.BlockedReason = nil
	user.BlockedAt = nil
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.UnblockUser update: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.UnblockUser reload: %w", err)
	}

	return updated, nil
}
