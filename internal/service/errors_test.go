package service

import (
	"errors"
	"testing"
)

func TestValidationError(t *testing.T) {
	ve := NewValidationError(FieldError{Field: "email", Reason: "required"})
	if ve.Error() != ErrInvalidInput.Error() {
		t.Fatalf("error: got %q", ve.Error())
	}
	if !errors.Is(ve, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput unwrap")
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Field != "email" {
		t.Fatalf("fields: %+v", ve.Fields)
	}
}

func TestRateLimitError(t *testing.T) {
	rl := &RateLimitError{Info: RateLimitInfo{Limit: 5, Remaining: 2, ResetSeconds: 10}}
	if rl.Error() != ErrRateLimited.Error() {
		t.Fatalf("error: got %q", rl.Error())
	}
	if !errors.Is(rl, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited unwrap")
	}
	if rl.Info.Limit != 5 || rl.Info.Remaining != 2 || rl.Info.ResetSeconds != 10 {
		t.Fatalf("info: %+v", rl.Info)
	}
}
