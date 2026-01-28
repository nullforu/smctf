package repo

import (
	"database/sql"
	"errors"
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

func TestMapNotFound(t *testing.T) {
	err := mapNotFound(nil)
	if err != nil {
		t.Errorf("expected nil for nil error, got %v", err)
	}

	originalErr := sql.ErrNoRows
	mapped := mapNotFound(originalErr)
	if mapped != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", mapped)
	}

	genericErr := errors.New("some error")
	mappedGeneric := mapNotFound(genericErr)
	if mappedGeneric != genericErr {
		t.Errorf("expected original error, got %v", mappedGeneric)
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
