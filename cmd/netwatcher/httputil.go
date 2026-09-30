package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func splitHostPort(target string) (string, int, bool) {
	if !strings.Contains(target, ":") {
		return "", 0, false
	}
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, false
	}
	port, err := parsePort(portStr)
	if err != nil {
		return "", 0, false
	}
	return host, port, true
}

type apiError struct {
	Error string `json:"error"`
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}

	type healthDep struct {
		Enabled bool   `json:"enabled"`
		Healthy bool   `json:"healthy"`
		Driver  string `json:"driver,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type healthResponse struct {
		OK          bool      `json:"ok"`
		Time        time.Time `json:"time"`
		Database    healthDep `json:"database"`
		Redis       healthDep `json:"redis"`
		ObjectStore healthDep `json:"object_storage"`
	}

	resp := healthResponse{
		OK:   true,
		Time: time.Now().UTC(),
		Database: healthDep{
			Enabled: appStore != nil,
		},
		Redis: healthDep{
			Enabled: appCache != nil,
		},
		ObjectStore: healthDep{
			Enabled: appObjectStore != nil,
		},
	}

	if appStore != nil {
		resp.Database.Driver = appStore.Driver()
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := appStore.Ping(ctx)
		cancel()
		resp.Database.Healthy = err == nil
		if err != nil {
			resp.Database.Error = err.Error()
			resp.OK = false
		}
	}
	if appCache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := appCache.Ping(ctx)
		cancel()
		resp.Redis.Healthy = err == nil
		if err != nil {
			resp.Redis.Error = err.Error()
			resp.OK = false
		}
	}
	if appObjectStore != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		err := appObjectStore.Ping(ctx)
		cancel()
		resp.ObjectStore.Healthy = err == nil
		if err != nil {
			resp.ObjectStore.Error = err.Error()
			resp.OK = false
		}
	}

	status := http.StatusOK
	if !resp.OK {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, resp)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty body")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return fmt.Errorf("request body too large")
		}
		return err
	}
	var extra struct{}
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
