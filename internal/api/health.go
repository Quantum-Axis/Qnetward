package api

import (
	"context"
	"net/http"
	"time"
)

func newBackendFallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":      true,
				"service": "qnetward-api",
				"ui":      "serve the Next.js frontend separately or use the frontend-primary Docker stack",
			})
			return
		}
		writeJSON(w, http.StatusNotFound, apiError{Error: "not found"})
	})
}

func (srv *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
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
			Enabled: srv.store != nil,
		},
		Redis: healthDep{
			Enabled: srv.cache != nil,
		},
		ObjectStore: healthDep{
			Enabled: srv.objects != nil,
		},
	}

	if srv.store != nil {
		resp.Database.Driver = srv.store.Driver()
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := srv.store.Ping(ctx)
		cancel()
		resp.Database.Healthy = err == nil
		if err != nil {
			resp.Database.Error = err.Error()
			resp.OK = false
		}
	}
	if srv.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := srv.cache.Ping(ctx)
		cancel()
		resp.Redis.Healthy = err == nil
		if err != nil {
			resp.Redis.Error = err.Error()
			resp.OK = false
		}
	}
	if srv.objects != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		err := srv.objects.Ping(ctx)
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
