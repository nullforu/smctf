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

type TeamService struct {
	teamRepo *repo.TeamRepo
}

func NewTeamService(teamRepo *repo.TeamRepo) *TeamService {
	return &TeamService{teamRepo: teamRepo}
}

func (s *TeamService) CreateTeam(ctx context.Context, name string) (*models.Team, error) {
	name = strings.TrimSpace(name)
	validator := newFieldValidator()
	validator.Required("name", name)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	team := &models.Team{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}

		return nil, fmt.Errorf("team.CreateTeam: %w", err)
	}

	return team, nil
}

func (s *TeamService) ListTeams(ctx context.Context) ([]models.TeamSummary, error) {
	rows, err := s.teamRepo.ListWithStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("team.ListTeams: %w", err)
	}

	return rows, nil
}

func (s *TeamService) GetTeam(ctx context.Context, id int64) (*models.TeamSummary, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	team, err := s.teamRepo.GetStats(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}

		return nil, fmt.Errorf("team.GetTeam: %w", err)
	}

	return team, nil
}

func (s *TeamService) ListMembers(ctx context.Context, id int64) ([]models.TeamMember, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.teamRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("team.ListMembers lookup: %w", err)
	}

	rows, err := s.teamRepo.ListMembers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team.ListMembers: %w", err)
	}

	return rows, nil
}

func (s *TeamService) ListSolvedChallenges(ctx context.Context, id int64) ([]models.TeamSolvedChallenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.teamRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}

		return nil, fmt.Errorf("team.ListSolvedChallenges lookup: %w", err)
	}

	rows, err := s.teamRepo.ListSolvedChallenges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team.ListSolvedChallenges: %w", err)
	}

	return rows, nil
}
