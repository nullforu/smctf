package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HMACFlag(secret, flag string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(flag))

	return hex.EncodeToString(h.Sum(nil))
}

func SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	return hmac.Equal([]byte(a), []byte(b))
}
