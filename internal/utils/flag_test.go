package utils

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashFlagAndCheck(t *testing.T) {
	flag := "flag{test}"
	hash1, err := HashFlag(flag, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashFlag failed: %v", err)
	}

	hash2, err := HashFlag(flag, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashFlag failed: %v", err)
	}

	if hash1 == hash2 {
		t.Fatalf("expected different hashes for same flag")
	}

	if !CheckFlag(hash1, flag) {
		t.Fatalf("expected CheckFlag to match")
	}

	if CheckFlag(hash1, "different") {
		t.Fatalf("expected CheckFlag to fail")
	}

	if CheckFlag("invalid-hash", flag) {
		t.Fatalf("expected CheckFlag to fail for invalid hash")
	}
}
