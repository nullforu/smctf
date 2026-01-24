package db

import (
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

const pgUniqueViolation = "23505"

func IsUniqueViolation(err error) bool {
	var pgerr pgdriver.Error
	if errors.As(err, &pgerr) {
		return pgerr.Field('C') == pgUniqueViolation
	}
	return false
}
