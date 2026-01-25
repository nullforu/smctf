package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseLimitQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?limit=10", nil)
	if got := parseLimitQuery(c, 50, 200); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?limit=500", nil)
	if got := parseLimitQuery(c, 50, 200); got != 200 {
		t.Fatalf("expected 200, got %d", got)
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?limit=0", nil)
	if got := parseLimitQuery(c, 50, 200); got != 50 {
		t.Fatalf("expected default 50, got %d", got)
	}
}

func TestParseIntervalQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?interval=15", nil)
	if got, err := parseIntervalQuery(c, 10); err != nil || got != 15 {
		t.Fatalf("expected 15, got %d err %v", got, err)
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?interval=-1", nil)
	if _, err := parseIntervalQuery(c, 10); err == nil {
		t.Fatalf("expected error for invalid interval")
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	if got, err := parseIntervalQuery(c, 10); err != nil || got != 10 {
		t.Fatalf("expected default 10, got %d err %v", got, err)
	}
}

func TestParseIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "123"}}
	if got, ok := parseIDParam(c, "id"); !ok || got != 123 {
		t.Fatalf("expected 123 ok, got %d ok %v", got, ok)
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "0"}}
	if _, ok := parseIDParam(c, "id"); ok {
		t.Fatalf("expected invalid id")
	}
}
