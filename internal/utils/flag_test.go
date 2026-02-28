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

	ok, err := CheckFlag(hash1, flag)
	if err != nil || !ok {
		t.Fatalf("expected CheckFlag to match, err=%v", err)
	}

	ok, err = CheckFlag(hash1, "different")
	if err != nil || ok {
		t.Fatalf("expected CheckFlag to fail")
	}

	ok, err = CheckFlag("invalid-hash", flag)
	if err == nil || ok {
		t.Fatalf("expected CheckFlag to error for invalid hash")
	}
}
