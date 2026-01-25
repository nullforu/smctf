package utils

import "testing"

func TestHMACFlagAndCompare(t *testing.T) {
	secret := "secret"
	flag := "flag{test}"
	hash1 := HMACFlag(secret, flag)
	hash2 := HMACFlag(secret, flag)
	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash")
	}
	if !SecureCompare(hash1, hash2) {
		t.Fatalf("expected secure compare to match")
	}
	if SecureCompare(hash1, "different") {
		t.Fatalf("expected secure compare to fail")
	}
}
