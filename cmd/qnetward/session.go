package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"qnetward/internal/store"
)

func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return store.User{}, false
	}
	user, _, err := currentSessionUser(r)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) || errors.Is(err, http.ErrNoCookie) {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "authentication required"})
			return store.User{}, false
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return store.User{}, false
	}
	return user, true
}

func requirePlatformAdmin(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if !isPlatformAdmin(user) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "platform admin access required"})
		return store.User{}, false
	}
	return user, true
}

func currentSessionState(r *http.Request) (store.User, string, int64, error) {
	tokenHash, err := sessionTokenHashFromRequest(r)
	if err != nil {
		return store.User{}, "", 0, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session, err := appStore.GetSessionDetailsByToken(ctx, tokenHash)
	if err != nil {
		return store.User{}, "", 0, err
	}
	return session.User, tokenHash, session.ActiveWorkspaceID, nil
}

func currentSessionUser(r *http.Request) (store.User, string, error) {
	user, tokenHash, _, err := currentSessionState(r)
	if err != nil {
		return store.User{}, "", err
	}
	return user, tokenHash, nil
}

func sessionTokenHashFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", http.ErrNoCookie
	}
	return hashSessionToken(token), nil
}

func issueSession(w http.ResponseWriter, r *http.Request, ctx context.Context, user store.User) error {
	token, tokenHash, err := generateSessionToken()
	if err != nil {
		return fmt.Errorf("failed to create session token")
	}
	expiresAt := time.Now().Add(defaultSessionTTL)
	if err := appStore.CreateSession(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return err
	}
	setSessionCookie(w, r, token, expiresAt)
	return nil
}

func isPlatformAdmin(user store.User) bool {
	if len(appAdminEmails) == 0 {
		return false
	}
	_, ok := appAdminEmails[strings.ToLower(strings.TrimSpace(user.Email))]
	return ok
}

func platformRoleForUser(user store.User) string {
	if isPlatformAdmin(user) {
		return "admin"
	}
	return "user"
}

func resolveActiveWorkspace(ctx context.Context, user store.User, activeWorkspaceID int64) (store.WorkspaceContext, error) {
	if activeWorkspaceID > 0 {
		workspace, err := appStore.GetWorkspaceContextByWorkspaceIDAndUserID(ctx, activeWorkspaceID, user.ID)
		if err == nil {
			return workspace, nil
		}
		if !errors.Is(err, store.ErrWorkspaceMemberNotFound) {
			return store.WorkspaceContext{}, err
		}
	}

	workspace, err := appStore.GetWorkspaceContextByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return appStore.EnsureDefaultWorkspaceForUser(ctx, user)
		}
		return store.WorkspaceContext{}, err
	}
	return workspace, nil
}

func generateSessionToken() (string, string, error) {
	return generateOpaqueToken()
}

func generatePasswordResetToken() (string, string, error) {
	return generateOpaqueToken()
}

func generateVerificationCode() (string, string, error) {
	const digits = "0123456789"
	buf := make([]byte, 6)
	for i := range buf {
		n, err := rand.Int(rand.Reader, bigTen)
		if err != nil {
			return "", "", err
		}
		buf[i] = digits[n.Int64()]
	}
	code := string(buf)
	return code, hashOpaqueToken(code), nil
}

func generateOpaqueToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashOpaqueToken(token), nil
}

func hashSessionToken(token string) string {
	return hashOpaqueToken(token)
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var bigTen = func() *big.Int {
	return big.NewInt(10)
}()

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func clearLegacyDemoCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "qnetward_demo",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func clearLegacyPreviewCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "qnetward_preview",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
