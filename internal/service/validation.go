package service

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const bcryptInputMaxBytes = 72

// nameMaxLen bounds username, team name, and division name to 10 characters
// each. Combined as "division_team_username" (10+10+10 + two underscores) this
// fits Discord's 32-character nickname limit exactly.
const nameMaxLen = 10

type fieldValidator struct {
	fields []FieldError
}

func newFieldValidator() *fieldValidator {
	return &fieldValidator{fields: make([]FieldError, 0, 4)}
}

func (v *fieldValidator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.fields = append(v.fields, FieldError{Field: field, Reason: "required"})
	}
}

func (v *fieldValidator) NonNegative(field string, value int) {
	if value < 0 {
		v.fields = append(v.fields, FieldError{Field: field, Reason: "must be >= 0"})
	}
}

func (v *fieldValidator) PositiveID(field string, value int64) {
	if value <= 0 {
		v.fields = append(v.fields, FieldError{Field: field, Reason: "invalid"})
	}
}

func (v *fieldValidator) Email(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	if _, err := mail.ParseAddress(value); err != nil {
		v.fields = append(v.fields, FieldError{Field: field, Reason: "invalid format"})
	}
}

// Snowflake validates an optional Discord ID: empty is allowed, otherwise the
// value must be a numeric snowflake (1-32 digits).
func (v *fieldValidator) Snowflake(field, value string) {
	if value == "" {
		return
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			v.fields = append(v.fields, FieldError{Field: field, Reason: "must be a numeric Discord ID"})
			return
		}
	}

	if len(value) > 32 {
		v.fields = append(v.fields, FieldError{Field: field, Reason: "must be a numeric Discord ID"})
	}
}

func (v *fieldValidator) MaxLen(field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		v.fields = append(v.fields, FieldError{Field: field, Reason: fmt.Sprintf("max length is %d characters", max)})
	}
}

func (v *fieldValidator) MaxBytes(field, value string, max int) {
	if len(value) > max {
		v.fields = append(v.fields, FieldError{Field: field, Reason: fmt.Sprintf("max bytes is %d", max)})
	}
}

func (v *fieldValidator) Error() error {
	if len(v.fields) == 0 {
		return nil
	}

	return NewValidationError(v.fields...)
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func normalizeTrim(value string) string {
	return strings.TrimSpace(value)
}

func normalizeDiscordID(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func derefOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := normalizeTrim(*value)
	return &normalized
}
