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

	// Verify it's a valid bcrypt hash
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Errorf("invalid bcrypt hash prefix: %s", hash)
	}
}

func TestHashPassword_DifferentCosts(t *testing.T) {
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

			// Verify the hash works with CheckPassword
			if !CheckPassword(hash, password) {
				t.Error("CheckPassword should return true for correct password")
			}
		})
	}
}

func TestHashPassword_InvalidCost(t *testing.T) {
	password := "test-password"

	// Only test cost too high, as bcrypt automatically adjusts low costs
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

	// Test correct password
	if !CheckPassword(hash, password) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Test incorrect password
	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword should return false for incorrect password")
	}

	// Test empty password
	if CheckPassword(hash, "") {
		t.Error("CheckPassword should return false for empty password")
	}

	// Test invalid hash
	if CheckPassword("invalid-hash", password) {
		t.Error("CheckPassword should return false for invalid hash")
	}

	// Test empty hash
	if CheckPassword("", password) {
		t.Error("CheckPassword should return false for empty hash")
	}
}

func TestHashPassword_SamePasswordDifferentHashes(t *testing.T) {
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

	// Hashes should be different (bcrypt uses random salt)
	if hash1 == hash2 {
		t.Error("hashes should be different due to random salt")
	}

	// Both hashes should verify the same password
	if !CheckPassword(hash1, password) {
		t.Error("first hash should verify password")
	}

	if !CheckPassword(hash2, password) {
		t.Error("second hash should verify password")
	}
}

func TestCheckPassword_CaseSensitive(t *testing.T) {
	password := "TestPassword123"
	cost := bcrypt.DefaultCost

	hash, err := HashPassword(password, cost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Check case sensitivity
	if CheckPassword(hash, "testpassword123") {
		t.Error("CheckPassword should be case sensitive")
	}

	if CheckPassword(hash, "TESTPASSWORD123") {
		t.Error("CheckPassword should be case sensitive")
	}
}
