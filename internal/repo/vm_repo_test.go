package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
	vmpkg "smctf/internal/vm"
)

func createVMRow(t *testing.T, vmRepo *VMRepo, userID, challengeID int64, vmID, status string, createdAt time.Time) *models.VM {
	t.Helper()
	row := &models.VM{
		UserID:       userID,
		ChallengeID:  challengeID,
		VMID:         vmID,
		Status:       status,
		Ports:        vmpkg.PortMappings{{HostPort: 10000, ContainerPort: 31337, Protocol: "tcp"}},
		TTLExpiresAt: ptrTime(createdAt.Add(time.Hour)),
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	if err := vmRepo.Create(context.Background(), row); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	return row
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestVMRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmcrud@example.com", "vmcrud", "pass", models.UserRole)
	challenge := createChallenge(t, env, "VM CRUD", 100, "flag{vmcrud}", true)

	now := time.Now().UTC()
	created := createVMRow(t, vmRepo, user.ID, challenge.ID, "vm-crud-1", "Pending", now)

	got, err := vmRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByUserAndChallenge: %v", err)
	}

	if got.VMID != created.VMID || got.Username != user.Username || got.ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected vm row: %+v", got)
	}

	got.Status = "Running"
	lastErr := "none"
	got.LastError = &lastErr
	if err := vmRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := vmRepo.GetByVMID(context.Background(), created.VMID)
	if err != nil {
		t.Fatalf("GetByVMID: %v", err)
	}

	if updated.Status != "Running" {
		t.Fatalf("expected updated status Running, got %s", updated.Status)
	}

	if err := vmRepo.Delete(context.Background(), updated); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), created.VMID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestVMRepoListAndCountByUser(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmlist@example.com", "vmlist", "pass", models.UserRole)
	challenge1 := createChallenge(t, env, "VM List 1", 100, "flag{vmlist1}", true)
	challenge2 := createChallenge(t, env, "VM List 2", 100, "flag{vmlist2}", true)

	now := time.Now().UTC()
	createVMRow(t, vmRepo, user.ID, challenge1.ID, "vm-old", "Running", now.Add(-time.Minute))
	createVMRow(t, vmRepo, user.ID, challenge2.ID, "vm-new", "Pending", now)

	list, err := vmRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 vm rows, got %d", len(list))
	}

	if list[0].VMID != "vm-new" {
		t.Fatalf("expected newest row first, got %+v", list)
	}

	count, err := vmRepo.CountByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestVMRepoListAdmin(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)
	user := createUserWithNewTeam(t, env, "vmadmin@example.com", "vmadmin", "pass", models.UserRole)
	challenge := createChallenge(t, env, "VM Admin", 300, "flag{vmadmin}", true)

	createVMRow(t, vmRepo, user.ID, challenge.ID, "vm-admin-1", "Running", time.Now().UTC())

	rows, err := vmRepo.ListAdmin(context.Background())
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].VMID != "vm-admin-1" || rows[0].Username != user.Username || rows[0].ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected admin row: %+v", rows[0])
	}
}

func TestVMRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	vmRepo := NewVMRepo(env.db)

	if _, err := vmRepo.GetByVMID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := vmRepo.GetByUserAndChallenge(context.Background(), 999, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
