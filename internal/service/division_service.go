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

const reservedAdminDivisionName = "Admin"

type DivisionService struct {
	divisionRepo *repo.DivisionRepo
}

type DivisionExportItem struct {
	ID                       int64     `json:"id"`
	Name                     string    `json:"name"`
	DiscordRoleID            *string   `json:"discord_role_id,omitempty"`
	DiscordAnnounceChannelID *string   `json:"discord_announce_channel_id,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
}

type DivisionExportBundle struct {
	Version      int                  `json:"version"`
	ExportedAt   time.Time            `json:"exported_at"`
	RequestedIDs []int64              `json:"requested_ids,omitempty"`
	Divisions    []DivisionExportItem `json:"divisions"`
}

func NewDivisionService(divisionRepo *repo.DivisionRepo) *DivisionService {
	return &DivisionService{divisionRepo: divisionRepo}
}

func (s *DivisionService) CreateDivision(ctx context.Context, name string, discordRoleID, discordAnnounceChannelID *string) (*models.Division, error) {
	name = strings.TrimSpace(name)
	roleID := normalizeDiscordID(discordRoleID)
	channelID := normalizeDiscordID(discordAnnounceChannelID)

	validator := newFieldValidator()
	validator.Required("name", name)
	validator.MaxLen("name", name, nameMaxLen)
	validator.Snowflake("discord_role_id", derefOrEmpty(roleID))
	validator.Snowflake("discord_announce_channel_id", derefOrEmpty(channelID))
	if err := validator.Error(); err != nil {
		return nil, err
	}

	division := &models.Division{
		Name:                     name,
		DiscordRoleID:            roleID,
		DiscordAnnounceChannelID: channelID,
		CreatedAt:                time.Now().UTC(),
	}

	if err := s.divisionRepo.Create(ctx, division); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}
		return nil, fmt.Errorf("division.CreateDivision: %w", err)
	}

	return division, nil
}

func (s *DivisionService) UpdateDivision(ctx context.Context, id int64, name string, discordRoleID, discordAnnounceChannelID *string) (*models.Division, error) {
	name = strings.TrimSpace(name)
	roleID := normalizeDiscordID(discordRoleID)
	channelID := normalizeDiscordID(discordAnnounceChannelID)

	validator := newFieldValidator()
	validator.PositiveID("id", id)
	validator.Required("name", name)
	validator.MaxLen("name", name, nameMaxLen)
	validator.Snowflake("discord_role_id", derefOrEmpty(roleID))
	validator.Snowflake("discord_announce_channel_id", derefOrEmpty(channelID))
	if err := validator.Error(); err != nil {
		return nil, err
	}

	division, err := s.divisionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("division.UpdateDivision get: %w", err)
	}

	if strings.EqualFold(division.Name, reservedAdminDivisionName) && !strings.EqualFold(name, reservedAdminDivisionName) {
		return nil, NewValidationError(FieldError{Field: "name", Reason: "reserved"})
	}

	division.Name = name
	division.DiscordRoleID = roleID
	division.DiscordAnnounceChannelID = channelID

	if err := s.divisionRepo.Update(ctx, division); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}

		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("division.UpdateDivision: %w", err)
	}

	return division, nil
}

func (s *DivisionService) ListDivisions(ctx context.Context) ([]models.Division, error) {
	rows, err := s.divisionRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("division.ListDivisions: %w", err)
	}

	return rows, nil
}

func (s *DivisionService) GetDivision(ctx context.Context, id int64) (*models.Division, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	division, err := s.divisionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("division.GetDivision: %w", err)
	}

	return division, nil
}

func (s *DivisionService) ExportDivisions(ctx context.Context, ids []int64) (*DivisionExportBundle, error) {
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

	var (
		divisions []models.Division
		err       error
	)
	if len(normalizedIDs) == 0 {
		divisions, err = s.divisionRepo.List(ctx)
	} else {
		divisions, err = s.divisionRepo.ListByIDs(ctx, normalizedIDs)
		if err == nil && len(divisions) != len(normalizedIDs) {
			return nil, NewValidationError(FieldError{Field: "ids", Reason: "contains unknown division id"})
		}
	}
	if err != nil {
		return nil, fmt.Errorf("division.ExportDivisions: %w", err)
	}

	items := make([]DivisionExportItem, 0, len(divisions))
	for _, division := range divisions {
		items = append(items, DivisionExportItem{
			ID:                       division.ID,
			Name:                     division.Name,
			DiscordRoleID:            division.DiscordRoleID,
			DiscordAnnounceChannelID: division.DiscordAnnounceChannelID,
			CreatedAt:                division.CreatedAt,
		})
	}

	return &DivisionExportBundle{
		Version:      1,
		ExportedAt:   time.Now().UTC(),
		RequestedIDs: normalizedIDs,
		Divisions:    items,
	}, nil
}

func (s *DivisionService) ImportDivisions(ctx context.Context, bundle DivisionExportBundle) ([]models.Division, error) {
	if bundle.Version != 1 {
		return nil, NewValidationError(FieldError{Field: "version", Reason: "unsupported"})
	}
	if len(bundle.Divisions) == 0 {
		return nil, NewValidationError(FieldError{Field: "divisions", Reason: "required"})
	}

	validator := newFieldValidator()
	items := make([]models.Division, 0, len(bundle.Divisions))
	seenIDs := make(map[int64]struct{}, len(bundle.Divisions))
	seenNames := make(map[string]struct{}, len(bundle.Divisions))
	for i, item := range bundle.Divisions {
		fieldPrefix := fmt.Sprintf("divisions[%d]", i)
		name := strings.TrimSpace(item.Name)
		roleID := normalizeDiscordID(item.DiscordRoleID)
		channelID := normalizeDiscordID(item.DiscordAnnounceChannelID)

		validator.PositiveID(fieldPrefix+".id", item.ID)
		validator.Required(fieldPrefix+".name", name)
		validator.MaxLen(fieldPrefix+".name", name, nameMaxLen)
		validator.Snowflake(fieldPrefix+".discord_role_id", derefOrEmpty(roleID))
		validator.Snowflake(fieldPrefix+".discord_announce_channel_id", derefOrEmpty(channelID))

		if _, ok := seenIDs[item.ID]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".id", Reason: "duplicate"})
		}
		seenIDs[item.ID] = struct{}{}

		lowerName := strings.ToLower(name)
		if _, ok := seenNames[lowerName]; ok {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".name", Reason: "duplicate"})
		}
		seenNames[lowerName] = struct{}{}

		if _, err := s.divisionRepo.GetByName(ctx, name); err == nil {
			validator.fields = append(validator.fields, FieldError{Field: fieldPrefix + ".name", Reason: "duplicate"})
		} else if !errors.Is(err, repo.ErrNotFound) {
			return nil, fmt.Errorf("division.ImportDivisions lookup: %w", err)
		}

		items = append(items, models.Division{
			ID:                       item.ID,
			Name:                     name,
			DiscordRoleID:            roleID,
			DiscordAnnounceChannelID: channelID,
			CreatedAt:                item.CreatedAt.UTC(),
		})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	imported, err := s.divisionRepo.ImportDivisions(ctx, items)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}
		return nil, fmt.Errorf("division.ImportDivisions: %w", err)
	}

	return imported, nil
}
