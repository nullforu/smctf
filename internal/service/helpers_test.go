package service

import "testing"

func TestTrimTo(t *testing.T) {
	if got := trimTo("short", 10); got != "short" {
		t.Fatalf("unexpected: %s", got)
	}

	if got := trimTo("toolong", 4); got != "tool" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestIsRegistrationCode(t *testing.T) {
	if !isRegistrationCode("ABCDEFGHJKLMNPQ2") {
		t.Fatalf("expected valid code")
	}

	if isRegistrationCode("123") {
		t.Fatalf("expected invalid code")
	}

	if isRegistrationCode("invalid-key!") {
		t.Fatalf("expected invalid code")
	}

	if isRegistrationCode("ABCDEFGHJKLMNPQ0") {
		t.Fatalf("expected invalid code")
	}

	if isRegistrationCode("abcdefghjklmnpq2") {
		t.Fatalf("expected invalid code")
	}
}

func TestGenerateRegistrationCode(t *testing.T) {
	code, err := generateRegistrationCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(code) != 16 {
		t.Fatalf("expected code length 16, got %d", len(code))
	}

	if !isRegistrationCode(code) {
		t.Fatalf("expected registration code, got %s", code)
	}
}
