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

type TeamExportItem struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	DivisionName string    `json:"division_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type TeamExportBundle struct {
	Version      int              `json:"version"`
	ExportedAt   time.Time        `json:"exported_at"`
	RequestedIDs []int64          `json:"requested_ids,omitempty"`
	Teams        []TeamExportItem `json:"teams"`
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

func (s *TeamService) ExportTeams(ctx context.Context, ids []int64) (*TeamExportBundle, error) {
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

	rows, err := s.teamRepo.ListWithStats(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("team.ExportTeams: %w", err)
	}

	selected := rows
	if len(normalizedIDs) > 0 {
		byID := make(map[int64]models.TeamSummary, len(rows))
		for _, row := range rows {
			byID[row.ID] = row
		}

		selected = make([]models.TeamSummary, 0, len(normalizedIDs))
		for _, id := range normalizedIDs {
			row, ok := byID[id]
			if !ok {
				return nil, NewValidationError(FieldError{Field: "ids", Reason: "contains unknown team id"})
			}
			selected = append(selected, row)
		}
	}

	items := make([]TeamExportItem, 0, len(selected))
	for _, row := range selected {
		items = append(items, TeamExportItem{
			ID:           row.ID,
			Name:         row.Name,
			DivisionName: row.DivisionName,
			CreatedAt:    row.CreatedAt,
		})
	}

	return &TeamExportBundle{
		Version:      1,
		ExportedAt:   time.Now().UTC(),
		RequestedIDs: normalizedIDs,
		Teams:        items,
	}, nil
}

func (s *TeamService) ImportTeams(ctx context.Context, bundle TeamExportBundle) ([]models.Team, error) {
	if bundle.Version != 1 {
		return nil, NewValidationError(FieldError{Field: "version", Reason: "unsupported"})
	}
	if len(bundle.Teams) == 0 {
		return nil, NewValidationError(FieldError{Field: "teams", Reason: "required"})
	}

	validator := newFieldValidator()
	seenIDs := make(map[int64]struct{}, len(bundle.Teams))
	seenNames := make(map[string]struct{}, len(bundle.Teams))
	items := make([]models.Team, 0, len(bundle.Teams))

	for i, item := range bundle.Teams {
		fieldPrefix := fmt.Sprintf("teams[%d]", i)
		name := strings.TrimSpace(item.Name)
		divisionName := strings.TrimSpace(item.DivisionName)

		validator.PositiveID(fieldPrefix+".id", item.ID)
		validator.Required(fieldPrefix+".name", name)
		validator.Required(fieldPrefix+".division_name", divisionName)

		if _, ok := seenIDs[item.ID]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".id", Reason: "duplicate"})
		}
		seenIDs[item.ID] = struct{}{}

		lowerName := strings.ToLower(name)
		if _, ok := seenNames[lowerName]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".name", Reason: "duplicate"})
		}
		seenNames[lowerName] = struct{}{}

		if _, err := s.teamRepo.GetByName(ctx, name); err == nil {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".name", Reason: "duplicate"})
		} else if !errors.Is(err, repo.ErrNotFound) {
			return nil, fmt.Errorf("team.ImportTeams lookup team: %w", err)
		}

		division, err := s.divisionRepo.GetByName(ctx, divisionName)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".division_name", Reason: "not found"})
				continue
			}
			return nil, fmt.Errorf("team.ImportTeams lookup division: %w", err)
		}

		items = append(items, models.Team{
			ID:         item.ID,
			Name:       name,
			DivisionID: division.ID,
			CreatedAt:  item.CreatedAt.UTC(),
		})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	imported, err := s.teamRepo.ImportTeams(ctx, items)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}
		return nil, fmt.Errorf("team.ImportTeams: %w", err)
	}

	return imported, nil
}
