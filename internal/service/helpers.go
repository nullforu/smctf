package service

import (
	"crypto/rand"
	"strings"
)

const (
	registrationCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	registrationCodeLength   = 16
)

var registrationCodeAllowed = func() [256]bool {
	var allowed [256]bool
	for i := range len(registrationCodeAlphabet) {
		allowed[registrationCodeAlphabet[i]] = true
	}

	return allowed
}()

func trimTo(value string, max int) string {
	if len(value) <= max {
		return value
	}

	return value[:max]
}

func isRegistrationCode(value string) bool {
	if len(value) != registrationCodeLength {
		return false
	}

	for i := 0; i < len(value); i++ {
		if !registrationCodeAllowed[value[i]] {
			return false
		}
	}

	return true
}

func generateRegistrationCode() (string, error) {
	alphabet := registrationCodeAlphabet
	length := registrationCodeLength
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
