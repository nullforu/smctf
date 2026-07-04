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
