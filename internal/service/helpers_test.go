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

func TestIsSixDigitCode(t *testing.T) {
	if !isSixDigitCode("123456") {
		t.Fatalf("expected valid code")
	}

	if isSixDigitCode("12345") {
		t.Fatalf("expected invalid code")
	}

	if isSixDigitCode("1234567") {
		t.Fatalf("expected invalid code")
	}

	if isSixDigitCode("12a456") {
		t.Fatalf("expected invalid code")
	}
}

func TestGenerateRegistrationCode(t *testing.T) {
	code, err := generateRegistrationCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(code) != 6 {
		t.Fatalf("expected code length 6, got %d", len(code))
	}

	if !isSixDigitCode(code) {
		t.Fatalf("expected six digit code, got %s", code)
	}
}
