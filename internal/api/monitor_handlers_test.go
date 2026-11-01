package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qnetward/internal/store"
)

// A monitor created through the API is checked by the runner: up while the
// port answers, down once it stops, with each probe also landing in history.
func TestMonitorCheckTracksUpAndDown(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)
	workspaceID := testWorkspaceID(t, st, user.ID)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	body := fmt.Sprintf(`{"name":"Local API","protocol":"tcpping","target":"127.0.0.1","port":%d,"interval_seconds":60}`, port)
	req := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	testSrv.handleMonitors(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created store.Monitor
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}
	created.WorkspaceID = workspaceID
	created.UserID = user.ID

	dup := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(body))
	dup.Header.Set("Content-Type", "application/json")
	dup.AddCookie(cookie)
	dupRR := httptest.NewRecorder()
	testSrv.handleMonitors(dupRR, dup)
	if dupRR.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d: %s", dupRR.Code, dupRR.Body.String())
	}

	testSrv.checkMonitor(context.Background(), created)
	if got := listMonitor(t, cookie); got.Status != "up" || got.LastRttMs == nil {
		t.Fatalf("expected the monitor up with a round-trip time, got %#v", got)
	}
	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(workspaceID, "tcpping", "127.0.0.1", port, 10)
		return err == nil && len(points) == 3
	})

	_ = ln.Close()
	testSrv.checkMonitor(context.Background(), created)
	got := listMonitor(t, cookie)
	if got.Status != "down" || got.LastError == "" {
		t.Fatalf("expected the monitor down with an error, got %#v", got)
	}
	if len(got.RecentChecks) != 2 || got.Uptime24h == nil || *got.Uptime24h != 50 {
		t.Fatalf("expected two checks at 50%% uptime, got %d checks, uptime %v", len(got.RecentChecks), got.Uptime24h)
	}

	del := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/monitors?id=%d", created.ID), nil)
	del.AddCookie(cookie)
	delRR := httptest.NewRecorder()
	testSrv.handleMonitors(delRR, del)
	if delRR.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", delRR.Code, delRR.Body.String())
	}
}

func listMonitor(t *testing.T, cookie *http.Cookie) store.Monitor {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/monitors", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	testSrv.handleMonitors(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var monitors []store.Monitor
	if err := json.Unmarshal(rr.Body.Bytes(), &monitors); err != nil {
		t.Fatalf("decode monitors: %v", err)
	}
	if len(monitors) != 1 {
		t.Fatalf("expected one monitor, got %d", len(monitors))
	}
	return monitors[0]
}
