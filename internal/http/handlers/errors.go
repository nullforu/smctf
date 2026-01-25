package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type errorResponse struct {
	Error     string                 `json:"error"`
	Details   []service.FieldError   `json:"details,omitempty"`
	RateLimit *service.RateLimitInfo `json:"rate_limit,omitempty"`
}

func writeError(ctx *gin.Context, err error) {
	status, resp, headers := mapError(err)
	for key, value := range headers {
		ctx.Header(key, value)
	}
	ctx.JSON(status, resp)
}

func mapError(err error) (int, errorResponse, map[string]string) {
	status := http.StatusInternalServerError
	resp := errorResponse{Error: "internal error"}

	var ve *service.ValidationError
	if errors.As(err, &ve) {
		status = http.StatusBadRequest
		resp.Error = ve.Error()
		resp.Details = ve.Fields
		return status, resp, nil
	}

	var rl *service.RateLimitError
	if errors.As(err, &rl) {
		status = http.StatusTooManyRequests
		resp.Error = rl.Error()
		resp.RateLimit = &rl.Info

		headers := map[string]string{
			"X-RateLimit-Limit":     strconv.Itoa(rl.Info.Limit),
			"X-RateLimit-Remaining": strconv.Itoa(rl.Info.Remaining),
			"X-RateLimit-Reset":     strconv.Itoa(rl.Info.ResetSeconds),
		}

		return status, resp, headers
	}

	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
		resp.Error = service.ErrInvalidInput.Error()
		resp.Details = []service.FieldError{{Field: "request", Reason: "invalid"}}
	case errors.Is(err, service.ErrInvalidCreds):
		status = http.StatusUnauthorized
		resp.Error = service.ErrInvalidCreds.Error()
	case errors.Is(err, service.ErrUserExists):
		status = http.StatusConflict
		resp.Error = service.ErrUserExists.Error()
	case errors.Is(err, service.ErrChallengeNotFound):
		status = http.StatusNotFound
		resp.Error = service.ErrChallengeNotFound.Error()
	case errors.Is(err, service.ErrAlreadySolved):
		status = http.StatusConflict
		resp.Error = service.ErrAlreadySolved.Error()
	case errors.Is(err, service.ErrRateLimited):
		status = http.StatusTooManyRequests
		resp.Error = service.ErrRateLimited.Error()
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		status = http.StatusNotFound
		resp.Error = "not found"
	}

	return status, resp, nil
}

func writeBindError(ctx *gin.Context, err error) {
	fields := bindErrorDetails(err)
	if len(fields) == 0 {
		fields = []service.FieldError{{Field: "body", Reason: "invalid"}}
	}

	ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: fields})
}

func bindErrorDetails(err error) []service.FieldError {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make([]service.FieldError, 0, len(verrs))
		for _, fe := range verrs {
			field := strings.ToLower(fe.Field())
			fields = append(fields, service.FieldError{Field: field, Reason: fe.Tag()})
		}

		return fields
	}

	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		field := strings.ToLower(ute.Field)
		if field == "" {
			field = "body"
		}
		return []service.FieldError{{Field: field, Reason: "invalid type"}}
	}

	var se *json.SyntaxError
	if errors.As(err, &se) {
		return []service.FieldError{{Field: "body", Reason: "invalid json"}}
	}

	if errors.Is(err, io.EOF) {
		return []service.FieldError{{Field: "body", Reason: "empty"}}
	}
	return nil
}
