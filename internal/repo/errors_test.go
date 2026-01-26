package repo

import (
	"testing"
)

func TestWrapError(t *testing.T) {
	err := wrapError("test.Op", nil)
	if err != nil {
		t.Errorf("expected nil for nil error, got %v", err)
	}

	originalErr := ErrNotFound
	wrapped := wrapError("test.Op", originalErr)
	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}

	expectedMsg := "test.Op: record not found"
	if wrapped.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, wrapped.Error())
	}
}

func TestWrapNotFound(t *testing.T) {
	err := wrapNotFound("test.Op", nil)
	if err != nil {
		t.Errorf("expected nil for nil error, got %v", err)
	}

	originalErr := ErrNotFound
	wrapped := wrapNotFound("test.Op", originalErr)
	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}
}
