package service

import (
	"errors"
	"testing"
)

func TestFieldValidator(t *testing.T) {
	v := newFieldValidator()
	v.Required("email", "")
	v.Email("email", "bad@@")
	v.Required("username", " ")
	v.NonNegative("points", -1)
	v.PositiveID("challenge_id", 0)

	err := v.Error()

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if len(ve.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(ve.Fields))
	}
}

func TestNormalizeHelpers(t *testing.T) {
	if got := normalizeEmail("  USER@EXAMPLE.COM "); got != "user@example.com" {
		t.Fatalf("unexpected email: %s", got)
	}

	if got := normalizeTrim("  hi  "); got != "hi" {
		t.Fatalf("unexpected trim: %s", got)
	}

	if got := normalizeOptional(nil); got != nil {
		t.Fatalf("expected nil")
	}

	val := "  hello  "
	if got := normalizeOptional(&val); *got != "hello" {
		t.Fatalf("unexpected optional: %v", got)
	}
}
