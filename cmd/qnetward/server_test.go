package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewBackendFallbackHandler(t *testing.T) {
	handler := newBackendFallbackHandler()

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRR := httptest.NewRecorder()
	handler.ServeHTTP(rootRR, rootReq)
	if rootRR.Code != http.StatusOK {
		t.Fatalf("root: expected 200, got %d", rootRR.Code)
	}
	if !strings.Contains(rootRR.Body.String(), "qnetward-api") {
		t.Fatalf("root: unexpected body %q", rootRR.Body.String())
	}

	unknownReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	unknownRR := httptest.NewRecorder()
	handler.ServeHTTP(unknownRR, unknownReq)
	if unknownRR.Code != http.StatusNotFound {
		t.Fatalf("unknown: expected 404, got %d", unknownRR.Code)
	}
}
