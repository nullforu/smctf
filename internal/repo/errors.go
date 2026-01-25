package repo

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a query yields no rows.
var ErrNotFound = errors.New("record not found")

func wrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func mapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func wrapNotFound(op string, err error) error {
	if err == nil {
		return nil
	}
	return wrapError(op, mapNotFound(err))
}
