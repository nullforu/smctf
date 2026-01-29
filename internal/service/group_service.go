package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"
)

type GroupService struct {
	groupRepo *repo.GroupRepo
}

func NewGroupService(groupRepo *repo.GroupRepo) *GroupService {
	return &GroupService{groupRepo: groupRepo}
}

func (s *GroupService) CreateGroup(ctx context.Context, name string) (*models.Group, error) {
	name = strings.TrimSpace(name)
	validator := newFieldValidator()
	validator.Required("name", name)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	group := &models.Group{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.groupRepo.Create(ctx, group); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}

		return nil, fmt.Errorf("group.CreateGroup: %w", err)
	}

	return group, nil
}

func (s *GroupService) ListGroups(ctx context.Context) ([]models.GroupSummary, error) {
	rows, err := s.groupRepo.ListWithStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("group.ListGroups: %w", err)
	}

	return rows, nil
}

func (s *GroupService) GetGroup(ctx context.Context, id int64) (*models.GroupSummary, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	group, err := s.groupRepo.GetStats(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}

		return nil, fmt.Errorf("group.GetGroup: %w", err)
	}

	return group, nil
}

func (s *GroupService) ListMembers(ctx context.Context, id int64) ([]models.GroupMember, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.groupRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("group.ListMembers lookup: %w", err)
	}

	rows, err := s.groupRepo.ListMembers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("group.ListMembers: %w", err)
	}

	return rows, nil
}

func (s *GroupService) ListSolvedChallenges(ctx context.Context, id int64) ([]models.GroupSolvedChallenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.groupRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}

		return nil, fmt.Errorf("group.ListSolvedChallenges lookup: %w", err)
	}

	rows, err := s.groupRepo.ListSolvedChallenges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("group.ListSolvedChallenges: %w", err)
	}

	return rows, nil
}
