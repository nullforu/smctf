package service

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

func trimTo(value string, max int) string {
	if len(value) <= max {
		return value
	}

	return value[:max]
}

func isSixDigitCode(value string) bool {
	if len(value) != 6 {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func generateRegistrationCode() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}

	value := binary.BigEndian.Uint32(buf[:]) % 1000000

	return fmt.Sprintf("%06d", value), nil
}
