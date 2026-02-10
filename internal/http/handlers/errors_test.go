package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/go-playground/validator/v10"
)

func TestMapErrorValidation(t *testing.T) {
	ve := service.NewValidationError(service.FieldError{Field: "email", Reason: "required"})
	status, resp, headers := mapError(ve)

	if status != http.StatusBadRequest {
		t.Fatalf("status: got %d", status)
	}

	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("error: got %q", resp.Error)
	}

	if len(resp.Details) != 1 || resp.Details[0].Field != "email" {
		t.Fatalf("details: %+v", resp.Details)
	}

	if headers != nil {
		t.Fatalf("expected no headers")
	}
}

func TestMapErrorRateLimit(t *testing.T) {
	rl := &service.RateLimitError{Info: service.RateLimitInfo{Limit: 5, Remaining: 1, ResetSeconds: 10}}
	status, resp, headers := mapError(rl)

	if status != http.StatusTooManyRequests {
		t.Fatalf("status: got %d", status)
	}

	if resp.Error != service.ErrRateLimited.Error() {
		t.Fatalf("error: got %q", resp.Error)
	}

	if resp.RateLimit == nil || resp.RateLimit.Limit != 5 {
		t.Fatalf("rate limit: %+v", resp.RateLimit)
	}

	if headers["X-RateLimit-Limit"] != "5" || headers["X-RateLimit-Remaining"] != "1" || headers["X-RateLimit-Reset"] != "10" {
		t.Fatalf("headers: %+v", headers)
	}
}

func TestMapErrorSentinels(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		expect  string
		details int
	}{
		{service.ErrInvalidInput, http.StatusBadRequest, service.ErrInvalidInput.Error(), 1},
		{service.ErrInvalidCreds, http.StatusUnauthorized, service.ErrInvalidCreds.Error(), 0},
		{service.ErrUserExists, http.StatusConflict, service.ErrUserExists.Error(), 0},
		{service.ErrChallengeNotFound, http.StatusNotFound, service.ErrChallengeNotFound.Error(), 0},
		{service.ErrChallengeFileNotFound, http.StatusNotFound, service.ErrChallengeFileNotFound.Error(), 0},
		{service.ErrStorageUnavailable, http.StatusServiceUnavailable, service.ErrStorageUnavailable.Error(), 0},
		{service.ErrAlreadySolved, http.StatusConflict, service.ErrAlreadySolved.Error(), 0},
		{service.ErrRateLimited, http.StatusTooManyRequests, service.ErrRateLimited.Error(), 0},
		{service.ErrStackDisabled, http.StatusServiceUnavailable, service.ErrStackDisabled.Error(), 0},
		{service.ErrStackNotEnabled, http.StatusBadRequest, service.ErrStackNotEnabled.Error(), 0},
		{service.ErrStackLimitReached, http.StatusConflict, service.ErrStackLimitReached.Error(), 0},
		{service.ErrStackNotFound, http.StatusNotFound, service.ErrStackNotFound.Error(), 0},
		{service.ErrStackProvisionerDown, http.StatusServiceUnavailable, service.ErrStackProvisionerDown.Error(), 0},
		{service.ErrStackInvalidSpec, http.StatusBadRequest, service.ErrStackInvalidSpec.Error(), 0},
		{repo.ErrNotFound, http.StatusNotFound, "not found", 0},
	}

	for _, tc := range cases {
		status, resp, _ := mapError(tc.err)
		if status != tc.status {
			t.Fatalf("%v status: got %d", tc.err, status)
		}

		if resp.Error != tc.expect {
			t.Fatalf("%v error: got %q", tc.err, resp.Error)
		}

		if tc.details != 0 && len(resp.Details) != tc.details {
			t.Fatalf("%v details: got %d", tc.err, len(resp.Details))
		}
	}
}

func TestWriteErrorRateLimitHeaders(t *testing.T) {
	ctx, rec := newJSONContext(t, http.MethodGet, "/", nil)

	rl := &service.RateLimitError{Info: service.RateLimitInfo{Limit: 5, Remaining: 2, ResetSeconds: 10}}
	writeError(ctx, rl)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status: got %d", rec.Code)
	}

	if rec.Header().Get("X-RateLimit-Limit") != "5" {
		t.Fatalf("missing limit header")
	}
}

type bindTestPayload struct {
	Email string `validate:"required"`
}

func TestBindErrorDetails(t *testing.T) {
	v := validator.New()
	if err := v.Struct(bindTestPayload{}); err == nil {
		t.Fatalf("expected validation error")
	} else {
		fields := bindErrorDetails(err)
		if len(fields) != 1 || fields[0].Field != "email" || fields[0].Reason != "required" {
			t.Fatalf("fields: %+v", fields)
		}
	}

	fields := bindErrorDetails(&json.UnmarshalTypeError{Field: "Email"})
	if len(fields) != 1 || fields[0].Field != "email" || fields[0].Reason != "invalid type" {
		t.Fatalf("type fields: %+v", fields)
	}

	fields = bindErrorDetails(&json.SyntaxError{})
	if len(fields) != 1 || fields[0].Reason != "invalid json" {
		t.Fatalf("syntax fields: %+v", fields)
	}

	fields = bindErrorDetails(io.EOF)
	if len(fields) != 1 || fields[0].Reason != "empty" {
		t.Fatalf("eof fields: %+v", fields)
	}

	fields = bindErrorDetails(errors.New("other"))
	if fields != nil {
		t.Fatalf("expected nil fields")
	}
}

func TestWriteBindErrorFallback(t *testing.T) {
	ctx, rec := newJSONContext(t, http.MethodPost, "/", nil)
	writeBindError(ctx, errors.New("boom"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", rec.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("error: got %q", resp.Error)
	}

	if len(resp.Details) != 1 || resp.Details[0].Field != "body" {
		t.Fatalf("details: %+v", resp.Details)
	}
}

func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"":        "",
		"Email":   "email",
		"UserID":  "user_id",
		"Team1ID": "team1_id",
		"ABC":     "abc",
		"FieldA":  "field_a",
	}

	for input, expected := range cases {
		if got := toSnakeCase(input); got != expected {
			t.Fatalf("toSnakeCase(%q) = %q", input, got)
		}
	}
}

func TestBindErrorDetailsBodyFieldFallback(t *testing.T) {
	fields := bindErrorDetails(&json.UnmarshalTypeError{Field: ""})
	if len(fields) != 1 || fields[0].Field != "body" {
		t.Fatalf("expected body field fallback: %+v", fields)
	}
}

func TestWriteErrorDefault(t *testing.T) {
	ctx, rec := newJSONContext(t, http.MethodGet, "/", nil)
	writeError(ctx, errors.New("unknown"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d", rec.Code)
	}
}
