package service

import (
	"errors"
	"strings"
	"testing"
)

func TestFieldValidator(t *testing.T) {
	v := newFieldValidator()
	v.Required("email", "")
	v.Email("email", "bad@@")
	v.Required("username", " ")
	v.NonNegative("points", -1)
	v.PositiveID("challenge_id", 0)
	v.MaxBytes("password", strings.Repeat("a", 73), bcryptInputMaxBytes)
	v.MaxLen("username", strings.Repeat("a", nameMaxLen+1), nameMaxLen)

	err := v.Error()

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if len(ve.Fields) != 7 {
		t.Fatalf("expected 7 fields, got %d", len(ve.Fields))
	}
}

func TestMaxLenCountsRunesNotBytes(t *testing.T) {
	// Exactly nameMaxLen multi-byte (Korean) characters must pass; one more fails.
	ok := newFieldValidator()
	ok.MaxLen("username", strings.Repeat("가", nameMaxLen), nameMaxLen)
	if err := ok.Error(); err != nil {
		t.Fatalf("expected %d Korean chars to pass, got %v", nameMaxLen, err)
	}

	over := newFieldValidator()
	over.MaxLen("username", strings.Repeat("가", nameMaxLen+1), nameMaxLen)
	if err := over.Error(); err == nil {
		t.Fatalf("expected %d Korean chars to fail", nameMaxLen+1)
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
