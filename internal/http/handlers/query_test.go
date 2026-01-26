package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = gin.Params{{Key: "id", Value: "123"}}
	if got, ok := parseIDParam(ctx, "id"); !ok || got != 123 {
		t.Fatalf("expected 123 ok, got %d ok %v", got, ok)
	}

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = gin.Params{{Key: "id", Value: "0"}}
	if _, ok := parseIDParam(ctx, "id"); ok {
		t.Fatalf("expected invalid id")
	}
}
