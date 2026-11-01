package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qnetward/internal/store"
)

func TestHandleHistoryReturnsStoredPoints(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)

	now := time.Now()
	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, user.ID),
		UserID:      user.ID,
		Ts:          now,
		Protocol:    "ping",
		Target:      "example.com",
		Addr:        "93.184.216.34",
		Seq:         1,
		RttMs:       float64Ptr(12.5),
	})

	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(testWorkspaceID(t, st, user.ID), "ping", "example.com", 0, 10)
		return err == nil && len(points) == 1
	})

	req := httptest.NewRequest(http.MethodGet, "/api/history?type=ping&host=example.com&limit=10", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	testSrv.handleHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var points []store.HistoryPoint
	if err := json.Unmarshal(rr.Body.Bytes(), &points); err != nil {
		t.Fatalf("failed to decode points: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Seq != 1 {
		t.Fatalf("expected seq 1, got %d", points[0].Seq)
	}
}

func TestHandleHistoryIsWorkspaceScoped(t *testing.T) {
	st := installTestStore(t)
	userA, _ := issueTestSession(t, st)
	_, cookieB := issueTestSession(t, st)

	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, userA.ID),
		UserID:      userA.ID,
		Ts:          time.Now(),
		Protocol:    "ping",
		Target:      "example.com",
		Addr:        "93.184.216.34",
		Seq:         1,
		RttMs:       float64Ptr(8.5),
	})

	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(testWorkspaceID(t, st, userA.ID), "ping", "example.com", 0, 10)
		return err == nil && len(points) == 1
	})

	req := httptest.NewRequest(http.MethodGet, "/api/history?type=ping&host=example.com&limit=10", nil)
	req.AddCookie(cookieB)
	rr := httptest.NewRecorder()
	testSrv.handleHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var points []store.HistoryPoint
	if err := json.Unmarshal(rr.Body.Bytes(), &points); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("expected no cross-workspace history leakage, got %d point(s)", len(points))
	}
}

func TestHandleRecentHistoryTargetsReturnsStoredTargets(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)

	now := time.Now()
	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, user.ID),
		UserID:      user.ID,
		Ts:          now.Add(-2 * time.Minute),
		Protocol:    "ping",
		Target:      "one.example",
		Seq:         1,
		RttMs:       float64Ptr(9.5),
	})
	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, user.ID),
		UserID:      user.ID,
		Ts:          now.Add(-time.Minute),
		Protocol:    "tcpping",
		Target:      "two.example",
		Port:        443,
		Seq:         1,
		RttMs:       float64Ptr(13.2),
	})

	waitFor(t, 10*time.Second, func() bool {
		targets, err := st.QueryRecentHistoryTargets(testWorkspaceID(t, st, user.ID), 10)
		return err == nil && len(targets) == 2
	})

	req := httptest.NewRequest(http.MethodGet, "/api/history/recent-targets?limit=10", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	testSrv.handleRecentHistoryTargets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var targets []store.RecentHistoryTarget
	if err := json.Unmarshal(rr.Body.Bytes(), &targets); err != nil {
		t.Fatalf("decode recent targets: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 recent targets, got %d", len(targets))
	}
	if targets[0].Protocol != "tcpping" || targets[0].Target != "two.example" {
		t.Fatalf("unexpected first recent target: %#v", targets[0])
	}
}

func TestHandleHistoryNormalizesTCPHostPort(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)

	now := time.Now()
	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, user.ID),
		UserID:      user.ID,
		Ts:          now,
		Protocol:    "tcpping",
		Target:      "example.com",
		Port:        443,
		Addr:        "93.184.216.34",
		Seq:         1,
		RttMs:       float64Ptr(12.5),
	})

	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(testWorkspaceID(t, st, user.ID), "tcpping", "example.com", 443, 10)
		return err == nil && len(points) == 1
	})

	req := httptest.NewRequest(http.MethodGet, "/api/history?type=tcpping&host=example.com:443&limit=10", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	testSrv.handleHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var points []store.HistoryPoint
	if err := json.Unmarshal(rr.Body.Bytes(), &points); err != nil {
		t.Fatalf("failed to decode points: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
}

func TestHandleHistoryRejectsInvalidLimit(t *testing.T) {
	st := installTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/api/history?type=ping&host=example.com&limit=5000", nil)
	req.AddCookie(issueTestSessionCookie(t, st))
	rr := httptest.NewRecorder()

	testSrv.handleHistory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "limit must be") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestHandleHistoryExportRequiresPostAndIsWorkspaceScoped(t *testing.T) {
	st := installTestStore(t)
	userA, cookieA := issueTestSession(t, st)
	_, cookieB := issueTestSession(t, st)

	st.InsertAsync(store.ResultRecord{
		WorkspaceID: testWorkspaceID(t, st, userA.ID),
		UserID:      userA.ID,
		Ts:          time.Now(),
		Protocol:    "ping",
		Target:      "example.com",
		Addr:        "93.184.216.34",
		Seq:         1,
		RttMs:       float64Ptr(15.5),
	})

	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(testWorkspaceID(t, st, userA.ID), "ping", "example.com", 0, 10)
		return err == nil && len(points) == 1
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/export/history", nil)
	getReq.AddCookie(cookieA)
	getRR := httptest.NewRecorder()
	testSrv.handleHistoryExport(getRR, getReq)
	if getRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", getRR.Code)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/export/history", strings.NewReader(`{"type":"ping","host":"example.com","limit":10,"format":"json"}`))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.AddCookie(cookieB)
	postRR := httptest.NewRecorder()
	testSrv.handleHistoryExport(postRR, postReq)

	if postRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", postRR.Code, postRR.Body.String())
	}
	if got := strings.TrimSpace(postRR.Body.String()); got != "[]" {
		t.Fatalf("expected empty export for another workspace, got %s", got)
	}
}

func TestHistoryCSVRowsIncludesHeader(t *testing.T) {
	rows := historyCSVRows([]store.HistoryPoint{{Ts: 123, Seq: 1, RttMs: float64Ptr(1.25)}})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if got := strings.Join(rows[0], ","); got != "timestamp_ms,sequence,rtt_ms,error" {
		t.Fatalf("unexpected header row: %q", got)
	}
}
