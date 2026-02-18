package stack

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMockClientDefaults(t *testing.T) {
	m := &MockClient{}

	if _, err := m.CreateStack(context.Background(), 80, "spec"); !errors.Is(err, ErrUnexpected) {
		t.Fatalf("expected ErrUnexpected, got %v", err)
	}

	if _, err := m.GetStackStatus(context.Background(), "id"); !errors.Is(err, ErrUnexpected) {
		t.Fatalf("expected ErrUnexpected, got %v", err)
	}

	if err := m.DeleteStack(context.Background(), "id"); !errors.Is(err, ErrUnexpected) {
		t.Fatalf("expected ErrUnexpected, got %v", err)
	}
}

func TestMockClientFunctions(t *testing.T) {
	m := &MockClient{}

	m.CreateStackFn = func(ctx context.Context, targetPort int, podSpec string) (*StackInfo, error) {
		if targetPort != 8080 || podSpec != "spec" {
			t.Fatalf("unexpected args: %d %s", targetPort, podSpec)
		}

		return &StackInfo{StackID: "stack-1"}, nil
	}

	m.GetStackStatusFn = func(ctx context.Context, stackID string) (*StackStatus, error) {
		if stackID != "stack-1" {
			t.Fatalf("unexpected stackID: %s", stackID)
		}

		return &StackStatus{StackID: stackID, Status: "running"}, nil
	}

	m.DeleteStackFn = func(ctx context.Context, stackID string) error {
		if stackID != "stack-1" {
			t.Fatalf("unexpected stackID: %s", stackID)
		}

		return nil
	}

	info, err := m.CreateStack(context.Background(), 8080, "spec")
	if err != nil {
		t.Fatalf("CreateStack: %v", err)
	}

	if info.StackID != "stack-1" {
		t.Fatalf("unexpected info: %+v", info)
	}

	status, err := m.GetStackStatus(context.Background(), "stack-1")
	if err != nil {
		t.Fatalf("GetStackStatus: %v", err)
	}

	if status.Status != "running" {
		t.Fatalf("unexpected status: %+v", status)
	}

	if err := m.DeleteStack(context.Background(), "stack-1"); err != nil {
		t.Fatalf("DeleteStack: %v", err)
	}
}

func TestProvisionerMockLifecycle(t *testing.T) {
	p := NewProvisionerMock()
	client := p.Client()

	info, err := client.CreateStack(context.Background(), 80, "spec")
	if err != nil {
		t.Fatalf("CreateStack: %v", err)
	}

	if info.StackID == "" || info.TargetPort != 80 {
		t.Fatalf("unexpected info: %+v", info)
	}

	status, err := client.GetStackStatus(context.Background(), info.StackID)
	if err != nil {
		t.Fatalf("GetStackStatus: %v", err)
	}

	if status.Status != "running" || status.TargetPort != 80 {
		t.Fatalf("unexpected status: %+v", status)
	}

	if err := client.DeleteStack(context.Background(), info.StackID); err != nil {
		t.Fatalf("DeleteStack: %v", err)
	}

	if err := client.DeleteStack(context.Background(), info.StackID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProvisionerMockOverrides(t *testing.T) {
	p := NewProvisionerMock()
	client := p.Client()

	p.SetCreateError(ErrUnavailable)
	if _, err := client.CreateStack(context.Background(), 80, "spec"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected create ErrUnavailable, got %v", err)
	}

	p.SetCreateError(nil)
	info, err := client.CreateStack(context.Background(), 80, "spec")
	if err != nil {
		t.Fatalf("CreateStack: %v", err)
	}

	p.SetStatus(info.StackID, "stopped")
	status, err := client.GetStackStatus(context.Background(), info.StackID)
	if err != nil {
		t.Fatalf("GetStackStatus: %v", err)
	}

	if status.Status != "stopped" {
		t.Fatalf("expected status stopped, got %s", status.Status)
	}

	p.SetStatusError(info.StackID, ErrUnavailable)
	if _, err := client.GetStackStatus(context.Background(), info.StackID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected status ErrUnavailable, got %v", err)
	}

	p.SetStatusError(info.StackID, nil)
	p.SetDeleteError(info.StackID, ErrUnavailable)
	if err := client.DeleteStack(context.Background(), info.StackID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected delete ErrUnavailable, got %v", err)
	}

	p.SetDeleteError(info.StackID, nil)
	if err := client.DeleteStack(context.Background(), info.StackID); err != nil {
		t.Fatalf("DeleteStack: %v", err)
	}

	if p.DeleteCount(info.StackID) == 0 {
		t.Fatalf("expected delete count")
	}

	if p.CreateCount() == 0 {
		t.Fatalf("expected create count")
	}

	p.AddStack(StackInfo{
		StackID:      "stack-extra",
		Status:       "running",
		TargetPort:   80,
		TTLExpiresAt: time.Now().UTC().Add(time.Hour),
	})

	if _, err := client.GetStackStatus(context.Background(), "stack-extra"); err != nil {
		t.Fatalf("GetStackStatus extra: %v", err)
	}
}
