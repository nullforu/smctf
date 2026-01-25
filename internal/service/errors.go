package service

import "errors"

var (
	ErrUserExists        = errors.New("user already exists")
	ErrInvalidCreds      = errors.New("invalid credentials")
	ErrInvalidInput      = errors.New("invalid input")
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrAlreadySolved     = errors.New("challenge already solved")
	ErrRateLimited       = errors.New("too many submissions")
)

type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return ErrInvalidInput.Error()
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

func NewValidationError(fields ...FieldError) *ValidationError {
	return &ValidationError{Fields: fields}
}

type RateLimitInfo struct {
	Limit        int `json:"limit"`
	Remaining    int `json:"remaining"`
	ResetSeconds int `json:"reset_seconds"`
}

type RateLimitError struct {
	Info RateLimitInfo
}

func (e *RateLimitError) Error() string {
	return ErrRateLimited.Error()
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}
