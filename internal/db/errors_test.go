package db

import (
	"errors"
	"testing"
)

func TestIsUniqueViolation(t *testing.T) {
	// Test with nil error
	if IsUniqueViolation(nil) {
		t.Error("expected IsUniqueViolation to return false for nil error")
	}

	// Test with generic error
	genericErr := errors.New("some error")
	if IsUniqueViolation(genericErr) {
		t.Error("expected IsUniqueViolation to return false for generic error")
	}
}
