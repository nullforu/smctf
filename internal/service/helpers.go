package service

import (
	"crypto/rand"
	"strings"
)

func trimTo(value string, max int) string {
	if len(value) <= max {
		return value
	}

	return value[:max]
}

func isRegistrationCode(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}

	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		return false
	}

	return true
}

func generateRegistrationCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const length = 16

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(length)
	for _, v := range buf {
		b.WriteByte(alphabet[int(v)%len(alphabet)])
	}

	return b.String(), nil
}
