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
	teamRepo     *repo.TeamRepo
	divisionRepo *repo.DivisionRepo
}

func NewTeamService(teamRepo *repo.TeamRepo, divisionRepo *repo.DivisionRepo) *TeamService {
	return &TeamService{teamRepo: teamRepo, divisionRepo: divisionRepo}
}

func (s *TeamService) CreateTeam(ctx context.Context, name string, divisionID int64) (*models.Team, error) {
	name = strings.TrimSpace(name)
	validator := newFieldValidator()
	validator.Required("name", name)
	validator.PositiveID("division_id", divisionID)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	team := &models.Team{
		Name:       name,
		DivisionID: divisionID,
		CreatedAt:  time.Now().UTC(),
	}

	if _, err := s.divisionRepo.GetByID(ctx, divisionID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, NewValidationError(FieldError{Field: "division_id", Reason: "not found"})
		}
		return nil, fmt.Errorf("team.CreateTeam division: %w", err)
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}

		return nil, fmt.Errorf("team.CreateTeam: %w", err)
	}

	return team, nil
}

func (s *TeamService) ListTeams(ctx context.Context, divisionID *int64) ([]models.TeamSummary, error) {
	if divisionID != nil {
		validator := newFieldValidator()
		validator.PositiveID("division_id", *divisionID)
		if err := validator.Error(); err != nil {
			return nil, err
		}
	}

	rows, err := s.teamRepo.ListWithStats(ctx, divisionID)
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
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("team.GetTeam: %w", err)
	}

	return team, nil
}

func (s *TeamService) ensureTeamExists(ctx context.Context, id int64, contextLabel string) error {
	if _, err := s.teamRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("%s lookup: %w", contextLabel, err)
	}
	return nil
}

func (s *TeamService) ListMembers(ctx context.Context, id int64) ([]models.TeamMember, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if err := s.ensureTeamExists(ctx, id, "team.ListMembers"); err != nil {
		return nil, err
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

	if err := s.ensureTeamExists(ctx, id, "team.ListSolvedChallenges"); err != nil {
		return nil, err
	}

	rows, err := s.teamRepo.ListSolvedChallenges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team.ListSolvedChallenges: %w", err)
	}

	return rows, nil
}
