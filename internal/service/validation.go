package service

import (
	"net/mail"
	"strings"
)

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

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := normalizeTrim(*value)
	return &normalized
}
