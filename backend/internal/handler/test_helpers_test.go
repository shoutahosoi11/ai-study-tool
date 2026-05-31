package handler

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
)

func assertHTTPStatus(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected HTTP %d error, got nil", want)
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != want {
		t.Fatalf("HTTP status = %d, want %d", httpErr.Code, want)
	}
	if want >= http.StatusInternalServerError && httpErr.Message == "" {
		t.Fatal("expected error message")
	}
}
