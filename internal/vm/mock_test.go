package vm

import (
	"context"
	"errors"
	"testing"
)

func TestMockClientCreateSandbox(t *testing.T) {
	t.Run("missing fn", func(t *testing.T) {
		client := &MockClient{}
		if _, err := client.CreateSandbox(context.Background(), "vm-1", "spec"); !errors.Is(err, ErrUnexpected) {
			t.Fatalf("expected ErrUnexpected, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		client := &MockClient{
			CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*Sandbox, error) {
				if id != "vm-1" || specYAML != "spec" {
					t.Fatalf("unexpected args id=%q spec=%q", id, specYAML)
				}
				return &Sandbox{ID: id}, nil
			},
		}

		sandbox, err := client.CreateSandbox(context.Background(), "vm-1", "spec")
		if err != nil {
			t.Fatalf("CreateSandbox: %v", err)
		}

		if sandbox.ID != "vm-1" {
			t.Fatalf("unexpected sandbox: %+v", sandbox)
		}
	})
}

func TestMockClientGetSandbox(t *testing.T) {
	t.Run("missing fn", func(t *testing.T) {
		client := &MockClient{}
		if _, err := client.GetSandbox(context.Background(), "vm-1"); !errors.Is(err, ErrUnexpected) {
			t.Fatalf("expected ErrUnexpected, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		client := &MockClient{
			GetSandboxFn: func(ctx context.Context, id string) (*Sandbox, error) {
				if id != "vm-1" {
					t.Fatalf("unexpected id %q", id)
				}
				return &Sandbox{ID: id}, nil
			},
		}

		sandbox, err := client.GetSandbox(context.Background(), "vm-1")
		if err != nil {
			t.Fatalf("GetSandbox: %v", err)
		}

		if sandbox.ID != "vm-1" {
			t.Fatalf("unexpected sandbox: %+v", sandbox)
		}
	})
}

func TestMockClientDeleteSandbox(t *testing.T) {
	t.Run("missing fn", func(t *testing.T) {
		client := &MockClient{}
		if err := client.DeleteSandbox(context.Background(), "vm-1"); !errors.Is(err, ErrUnexpected) {
			t.Fatalf("expected ErrUnexpected, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		client := &MockClient{
			DeleteSandboxFn: func(ctx context.Context, id string) error {
				called = true
				if id != "vm-1" {
					t.Fatalf("unexpected id %q", id)
				}
				return nil
			},
		}

		if err := client.DeleteSandbox(context.Background(), "vm-1"); err != nil {
			t.Fatalf("DeleteSandbox: %v", err)
		}

		if !called {
			t.Fatalf("expected DeleteSandboxFn to be called")
		}
	})
}

func TestNewOrchestratorMockLifecycle(t *testing.T) {
	orch := NewOrchestratorMock()
	client := orch.Client()

	sandbox, err := client.CreateSandbox(context.Background(), "vm-mock-1", "spec")
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	if sandbox.ID != "vm-mock-1" {
		t.Fatalf("unexpected sandbox id: %+v", sandbox)
	}

	if sandbox.Status.Phase == "" {
		t.Fatalf("expected phase to be set")
	}

	got, err := client.GetSandbox(context.Background(), "vm-mock-1")
	if err != nil {
		t.Fatalf("GetSandbox: %v", err)
	}

	if got.ID != "vm-mock-1" {
		t.Fatalf("unexpected sandbox: %+v", got)
	}

	if err := client.DeleteSandbox(context.Background(), "vm-mock-1"); err != nil {
		t.Fatalf("DeleteSandbox: %v", err)
	}

	if _, err := client.GetSandbox(context.Background(), "vm-mock-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
