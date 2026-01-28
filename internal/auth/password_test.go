package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := "test-password-123"
	cost := bcrypt.DefaultCost

	hash, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	if hash == password {
		t.Fatal("hash should not be equal to plain password")
	}

	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Errorf("invalid bcrypt hash prefix: %s", hash)
	}
}

func TestHashPasswordDifferentCosts(t *testing.T) {
	password := "test-password"

	tests := []struct {
		name string
		cost int
	}{
		{"min cost", bcrypt.MinCost},
		{"default cost", bcrypt.DefaultCost},
		{"cost 13", 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(password, tt.cost)
			if err != nil {
				t.Fatalf("HashPassword failed with cost %d: %v", tt.cost, err)
			}

			if hash == "" {
				t.Fatal("expected non-empty hash")
			}

			if !CheckPassword(hash, password) {
				t.Error("CheckPassword should return true for correct password")
			}
		})
	}
}

func TestHashPasswordInvalidCost(t *testing.T) {
	password := "test-password"

	_, err := HashPassword(password, bcrypt.MaxCost+1)
	if err == nil {
		t.Error("expected error for cost too high, got nil")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "correct-password"
	cost := bcrypt.DefaultCost

	hash, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(hash, password) {
		t.Error("CheckPassword should return true for correct password")
	}

	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword should return false for incorrect password")
	}

	if CheckPassword(hash, "") {
		t.Error("CheckPassword should return false for empty password")
	}

	if CheckPassword("invalid-hash", password) {
		t.Error("CheckPassword should return false for invalid hash")
	}

	if CheckPassword("", password) {
		t.Error("CheckPassword should return false for empty hash")
	}
}

func TestHashPasswordSamePasswordDifferentHashes(t *testing.T) {
	password := "test-password"
	cost := bcrypt.DefaultCost

	hash1, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("first HashPassword failed: %v", err)
	}

	hash2, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("second HashPassword failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("hashes should be different due to random salt")
	}

	if !CheckPassword(hash1, password) {
		t.Error("first hash should verify password")
	}

	if !CheckPassword(hash2, password) {
		t.Error("second hash should verify password")
	}
}

func TestCheckPasswordCaseSensitive(t *testing.T) {
	password := "TestPassword123"
	cost := bcrypt.DefaultCost

	hash, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if CheckPassword(hash, "testpassword123") {
		t.Error("CheckPassword should be case sensitive")
	}

	if CheckPassword(hash, "TESTPASSWORD123") {
		t.Error("CheckPassword should be case sensitive")
	}
}
