package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestCORS_AllowsConfiguredFrontendURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS("http://localhost:8081"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 1. Preflight OPTIONS request
	reqOptions, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	reqOptions.Header.Set("Origin", "http://localhost:8081")
	reqOptions.Header.Set("Access-Control-Request-Method", "POST")
	wOptions := httptest.NewRecorder()
	r.ServeHTTP(wOptions, reqOptions)

	if wOptions.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for preflight, got %d", wOptions.Code)
	}
	if origin := wOptions.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8081" {
		t.Errorf("expected origin http://localhost:8081, got %q", origin)
	}
	if creds := wOptions.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected allow credentials true, got %q", creds)
	}

	// 2. Standard GET request
	reqGet, _ := http.NewRequest(http.MethodGet, "/test", nil)
	reqGet.Header.Set("Origin", "http://localhost:8081")
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", wGet.Code)
	}
	if origin := wGet.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8081" {
		t.Errorf("expected origin http://localhost:8081, got %q", origin)
	}
}

