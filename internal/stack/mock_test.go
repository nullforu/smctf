package stack

import (
	"context"
	"errors"
	"testing"
)

func TestMockClient_Defaults(t *testing.T) {
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

func TestMockClient_Functions(t *testing.T) {
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
