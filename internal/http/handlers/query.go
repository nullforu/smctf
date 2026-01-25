package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func parseLimitQuery(c *gin.Context, def, max int) int {
	value := strings.TrimSpace(c.Query("limit"))
	if value == "" {
		return def
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func parseIntervalQuery(c *gin.Context, def int) (int, error) {
	value := strings.TrimSpace(c.Query("interval"))
	if value == "" {
		return def, nil
	}
	interval, err := strconv.Atoi(value)
	if err != nil || interval <= 0 {
		return 0, errors.New("invalid interval")
	}
	return interval, nil
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	value := strings.TrimSpace(c.Param(name))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
