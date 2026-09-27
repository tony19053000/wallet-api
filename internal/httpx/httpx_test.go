package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tony19053000/wallet-api/internal/httpx"
)

func TestFormatPaise(t *testing.T) {
	tests := []struct {
		paise    int64
		expected string
	}{
		{0, "₹0.00"},
		{50000, "₹500.00"},
		{12500000, "₹1,25,000.00"},
		{4830000, "₹48,300.00"},
		{-200, "-₹2.00"},
	}

	for _, tc := range tests {
		res := httpx.FormatPaise(tc.paise)
		if res != tc.expected {
			t.Errorf("FormatPaise(%d) = %s; want %s", tc.paise, res, tc.expected)
		}
	}
}

func TestJSONResponseHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.BadRequest(rec, "invalid_input", "Test message")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected application/json header, got %s", contentType)
	}
}
