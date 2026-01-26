package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func parseWindowQuery(ctx *gin.Context) (int, error) {
	value := strings.TrimSpace(ctx.Query("window"))
	if value == "" {
		return 0, nil
	}

	window, err := strconv.Atoi(value)
	if err != nil || window <= 0 {
		return 0, errors.New("invalid window")
	}

	return window, nil
}

func parseIDParam(ctx *gin.Context, name string) (int64, bool) {
	value := strings.TrimSpace(ctx.Param(name))
	if value == "" {
		return 0, false
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}
