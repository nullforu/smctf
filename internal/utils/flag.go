package utils

import "golang.org/x/crypto/bcrypt"

func HashFlag(flag string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(flag), cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckFlag(hash, flag string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(flag)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
