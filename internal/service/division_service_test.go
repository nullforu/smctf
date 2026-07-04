package service

import (
	"context"
	"errors"
	"testing"
)

func sp(s string) *string { return &s }

func TestDivisionServiceCreateListGet(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "", nil, nil); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "ThisNameIsWayTooLong", nil, nil); err == nil {
		t.Fatalf("expected max-length validation error")
	}

	division, err := env.divisionSvc.CreateDivision(context.Background(), "Alpha", sp("123456789012345678"), sp("987654321098765432"))
	if err != nil {
		t.Fatalf("create division: %v", err)
	}

	if division.DiscordRoleID == nil || *division.DiscordRoleID != "123456789012345678" {
		t.Fatalf("role id not stored: %+v", division.DiscordRoleID)
	}
	if division.DiscordAnnounceChannelID == nil || *division.DiscordAnnounceChannelID != "987654321098765432" {
		t.Fatalf("channel id not stored: %+v", division.DiscordAnnounceChannelID)
	}

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "Alpha", nil, nil); err == nil {
		t.Fatalf("expected duplicate error")
	}

	list, err := env.divisionSvc.ListDivisions(context.Background())
	if err != nil {
		t.Fatalf("list divisions: %v", err)
	}

	if len(list) < 2 {
		t.Fatalf("expected at least 2 divisions, got %d", len(list))
	}

	got, err := env.divisionSvc.GetDivision(context.Background(), division.ID)
	if err != nil {
		t.Fatalf("get division: %v", err)
	}

	if got.ID != division.ID || got.Name != division.Name {
		t.Fatalf("unexpected division: %+v", got)
	}
}

func TestDivisionServiceCreateRejectsInvalidSnowflake(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "Beta", sp("not-a-number"), nil); err == nil {
		t.Fatalf("expected snowflake validation error for role id")
	}

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "Beta", nil, sp("12x34")); err == nil {
		t.Fatalf("expected snowflake validation error for channel id")
	}
}

func TestDivisionServiceCreateBlankDiscordIDsStoreNil(t *testing.T) {
	env := setupServiceTest(t)

	division, err := env.divisionSvc.CreateDivision(context.Background(), "Gamma", sp("   "), sp(""))
	if err != nil {
		t.Fatalf("create division: %v", err)
	}

	if division.DiscordRoleID != nil || division.DiscordAnnounceChannelID != nil {
		t.Fatalf("blank discord ids should be nil, got %+v / %+v", division.DiscordRoleID, division.DiscordAnnounceChannelID)
	}
}

func TestDivisionServiceUpdate(t *testing.T) {
	env := setupServiceTest(t)

	division, err := env.divisionSvc.CreateDivision(context.Background(), "Delta", nil, nil)
	if err != nil {
		t.Fatalf("create division: %v", err)
	}

	updated, err := env.divisionSvc.UpdateDivision(context.Background(), division.ID, "Delta2", sp("111111111111111111"), sp("222222222222222222"))
	if err != nil {
		t.Fatalf("update division: %v", err)
	}

	if updated.Name != "Delta2" {
		t.Errorf("name = %q", updated.Name)
	}

	if updated.DiscordRoleID == nil || *updated.DiscordRoleID != "111111111111111111" {
		t.Errorf("role id = %+v", updated.DiscordRoleID)
	}

	// Persisted?
	got, err := env.divisionSvc.GetDivision(context.Background(), division.ID)
	if err != nil {
		t.Fatalf("get division: %v", err)
	}
	if got.Name != "Delta2" || got.DiscordAnnounceChannelID == nil || *got.DiscordAnnounceChannelID != "222222222222222222" {
		t.Fatalf("update not persisted: %+v", got)
	}

	// Clearing the discord config back to nil.
	cleared, err := env.divisionSvc.UpdateDivision(context.Background(), division.ID, "Delta2", nil, nil)
	if err != nil {
		t.Fatalf("clear update: %v", err)
	}

	if cleared.DiscordRoleID != nil || cleared.DiscordAnnounceChannelID != nil {
		t.Fatalf("expected cleared discord ids, got %+v / %+v", cleared.DiscordRoleID, cleared.DiscordAnnounceChannelID)
	}
}

func TestDivisionServiceUpdateValidation(t *testing.T) {
	env := setupServiceTest(t)

	division, err := env.divisionSvc.CreateDivision(context.Background(), "Epsilon", nil, nil)
	if err != nil {
		t.Fatalf("create division: %v", err)
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), 0, "Name", nil, nil); err == nil {
		t.Fatalf("expected invalid id error")
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), division.ID, "", nil, nil); err == nil {
		t.Fatalf("expected required name error")
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), division.ID, "Epsilon", sp("bad-id"), nil); err == nil {
		t.Fatalf("expected snowflake error")
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), 999999, "Missing", nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDivisionServiceUpdateDuplicateName(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.CreateDivision(context.Background(), "Zeta", nil, nil); err != nil {
		t.Fatalf("create Zeta: %v", err)
	}
	other, err := env.divisionSvc.CreateDivision(context.Background(), "Eta", nil, nil)
	if err != nil {
		t.Fatalf("create Eta: %v", err)
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), other.ID, "Zeta", nil, nil); err == nil {
		t.Fatalf("expected duplicate name error")
	}
}

func TestDivisionServiceUpdateCannotRenameReservedAdmin(t *testing.T) {
	env := setupServiceTest(t)

	admin, err := env.divisionSvc.CreateDivision(context.Background(), "Admin", nil, nil)
	if err != nil {
		t.Fatalf("create admin division: %v", err)
	}

	if _, err := env.divisionSvc.UpdateDivision(context.Background(), admin.ID, "Public", nil, nil); err == nil {
		t.Fatalf("expected rename of reserved admin division to be rejected")
	}

	updated, err := env.divisionSvc.UpdateDivision(context.Background(), admin.ID, "Admin", sp("123456789012345678"), nil)
	if err != nil {
		t.Fatalf("expected discord-only update to succeed, got %v", err)
	}

	if updated.DiscordRoleID == nil || *updated.DiscordRoleID != "123456789012345678" {
		t.Fatalf("role id not updated: %+v", updated.DiscordRoleID)
	}
}

func TestDivisionServiceGetDivisionErrors(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.divisionSvc.GetDivision(context.Background(), 0); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.divisionSvc.GetDivision(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
