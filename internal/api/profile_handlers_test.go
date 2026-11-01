package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeProfile(t *testing.T) {
	got, err := normalizeProfile(profileUpdateRequest{
		Bio:         "  Network engineer  ",
		Pronouns:    "they/them",
		Website:     "qnetward.example.com",
		Timezone:    "Asia/Dhaka",
		SocialLinks: []string{"https://github.com/example", "", "HTTPS://github.com/example", "x.com/example"},
	})
	if err != nil {
		t.Fatalf("normalizeProfile: %v", err)
	}
	if got.Bio != "Network engineer" {
		t.Errorf("bio = %q, want trimmed", got.Bio)
	}
	if got.Website != "https://qnetward.example.com" {
		t.Errorf("website = %q, want https:// added", got.Website)
	}
	if len(got.SocialLinks) != 2 || got.SocialLinks[1] != "https://x.com/example" {
		t.Errorf("social links = %v, want blanks and case-insensitive duplicates dropped", got.SocialLinks)
	}
}

func TestNormalizeProfileRejectsBadInput(t *testing.T) {
	cases := map[string]profileUpdateRequest{
		"long bio":         {Bio: strings.Repeat("a", maxBioLength+1)},
		"long pronouns":    {Pronouns: strings.Repeat("a", maxPronounsLength+1)},
		"bad scheme":       {Website: "javascript:alert(1)"},
		"ftp":              {Website: "ftp://example.com"},
		"credentials":      {Website: "https://user:pass@example.com"},
		"unknown timezone": {Timezone: "Mars/Olympus"},
		"local timezone":   {Timezone: "Local"},
		"too many links":   {SocialLinks: []string{"a.com", "b.com", "c.com", "d.com", "e.com"}},
	}
	for name, req := range cases {
		if _, err := normalizeProfile(req); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestAccountProfileRoundTrip(t *testing.T) {
	st := installTestStore(t)
	_, cookie := issueTestSession(t, st)

	get := func() profileResponse {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/account/profile", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		testSrv.handleAccountProfile(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("GET: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp profileResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return resp
	}

	if empty := get(); empty.Profile.Bio != "" || empty.Profile.SocialLinks == nil {
		t.Fatalf("expected an empty profile with a non-nil links slice, got %#v", empty.Profile)
	}

	body := `{"bio":"Runs the NOC","pronouns":"she/her","website":"example.com","timezone":"Asia/Dhaka","social_links":["github.com/noc"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/account/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	testSrv.handleAccountProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	saved := get().Profile
	if saved.Bio != "Runs the NOC" || saved.Timezone != "Asia/Dhaka" || saved.Website != "https://example.com" {
		t.Fatalf("unexpected saved profile: %#v", saved)
	}
	if len(saved.SocialLinks) != 1 || saved.SocialLinks[0] != "https://github.com/noc" {
		t.Fatalf("unexpected social links: %v", saved.SocialLinks)
	}
	if saved.UpdatedAt == nil {
		t.Fatal("expected updated_at to be set")
	}
}

func TestAccountProfileRejectsInvalidAndUnauthenticated(t *testing.T) {
	st := installTestStore(t)
	_, cookie := issueTestSession(t, st)

	req := httptest.NewRequest(http.MethodPut, "/api/account/profile", strings.NewReader(`{"timezone":"Nowhere/City"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	testSrv.handleAccountProfile(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid timezone: expected 400, got %d", rr.Code)
	}

	anon := httptest.NewRecorder()
	testSrv.handleAccountProfile(anon, httptest.NewRequest(http.MethodGet, "/api/account/profile", nil))
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: expected 401, got %d", anon.Code)
	}
}

func TestAccountProfileRejectsDemoSession(t *testing.T) {
	installTestStore(t)
	originalEnabled, originalName, originalEmail, originalCreatedAt := testSrv.demoEnabled, testSrv.demoName, testSrv.demoEmail, testSrv.demoCreatedAt
	t.Cleanup(func() {
		testSrv.demoEnabled, testSrv.demoName, testSrv.demoEmail, testSrv.demoCreatedAt = originalEnabled, originalName, originalEmail, originalCreatedAt
	})
	testSrv.demoEnabled = true
	testSrv.demoName = "Contributor Demo"
	testSrv.demoEmail = "demo-account-profile@qnetward.local"
	testSrv.demoCreatedAt = time.Now().UTC()

	loginRR := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/demo-login", strings.NewReader(`{}`))
	loginReq.Header.Set("Content-Type", "application/json")
	testSrv.handleAuthDemoLogin(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("demo login: expected 200, got %d", loginRR.Code)
	}
	cookie := extractAuthCookie(t, loginRR.Result())

	req := httptest.NewRequest(http.MethodPut, "/api/account/profile", strings.NewReader(`{"bio":"changed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	testSrv.handleAccountProfile(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("demo PUT: expected 403, got %d", rr.Code)
	}
}
