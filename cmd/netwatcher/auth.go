package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"netwatcher/internal/mailer"
	"netwatcher/internal/store"
)

type authRequest struct {
	Name            string `json:"name,omitempty"`
	FirstName       string `json:"first_name,omitempty"`
	LastName        string `json:"last_name,omitempty"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

type authForgotPasswordRequest struct {
	Email string `json:"email"`
}

type authResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

type authVerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type authUser struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type authSessionResponse struct {
	Authenticated bool      `json:"authenticated"`
	User          *authUser `json:"user,omitempty"`
	Mode          string    `json:"mode,omitempty"`
	ReadOnly      bool      `json:"read_only,omitempty"`
}

type authMessageResponse struct {
	OK                   bool      `json:"ok"`
	Message              string    `json:"message"`
	PreviewURL           string    `json:"preview_url,omitempty"`
	PreviewCode          string    `json:"preview_code,omitempty"`
	VerificationRequired bool      `json:"verification_required,omitempty"`
	Authenticated        bool      `json:"authenticated,omitempty"`
	User                 *authUser `json:"user,omitempty"`
	Email                string    `json:"email,omitempty"`
	Mode                 string    `json:"mode,omitempty"`
	ReadOnly             bool      `json:"read_only,omitempty"`
}

func handleAuthSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore != nil {
		user, _, err := currentSessionUser(r)
		if err == nil {
			mode := "authenticated"
			if isDemoUser(user) {
				mode = "demo"
			}
			writeJSON(w, http.StatusOK, authSessionResponse{
				Authenticated: true,
				User:          toAuthUser(user),
				Mode:          mode,
			})
			return
		}
		if !errors.Is(err, store.ErrSessionNotFound) && !errors.Is(err, http.ErrNoCookie) {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, authSessionResponse{Authenticated: false})
}

func handleAuthDemoLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !appDemoEnabled {
		writeJSON(w, http.StatusNotFound, apiError{Error: "demo access is disabled"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := getOrCreateDemoUser(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	clearLegacyDemoCookie(w, r)
	clearLegacyPreviewCookie(w, r)
	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:            true,
		Message:       "Demo access granted.",
		Authenticated: true,
		User:          toAuthUser(user),
		Mode:          "demo",
	})
}

func handleAuthSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	name, err := normalizeSignupDisplayName(req.Name, req.FirstName, req.LastName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-signup", email, rateLimitConfig{Limit: 5, Window: 15 * time.Minute}) {
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if err := validatePasswordConfirmation(req.Password, req.ConfirmPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to hash password"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredSessions(ctx)
	_ = appStore.DeleteExpiredEmailVerificationCodes(ctx)
	user, err := appStore.CreateUser(ctx, name, email, string(passwordHash), true)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiError{Error: "user already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		}
		return
	}

	resp, err := prepareVerificationChallenge(ctx, r, user)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-login", email, rateLimitConfig{Limit: 8, Window: 15 * time.Minute}) {
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "password is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredSessions(ctx)
	user, passwordHash, err := appStore.AuthenticateUser(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid email or password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid email or password"})
		return
	}
	if user.VerificationRequired {
		writeJSON(w, http.StatusForbidden, authMessageResponse{
			OK:                   false,
			Message:              "Email verification required. Enter the OTP sent to your inbox.",
			VerificationRequired: true,
			Email:                user.Email,
		})
		return
	}

	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, authSessionResponse{
		Authenticated: true,
		User:          toAuthUser(user),
	})
}

func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if tokenHash, err := sessionTokenHashFromRequest(r); err == nil && tokenHash != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		_ = appStore.DeleteSession(ctx, tokenHash)
		cancel()
	}
	clearSessionCookie(w, r)
	clearLegacyDemoCookie(w, r)
	clearLegacyPreviewCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleAuthVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authVerifyEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-verify-email", email, rateLimitConfig{Limit: 6, Window: 10 * time.Minute}) {
		return
	}
	code := normalizeVerificationCode(req.Code)
	if code == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "verification code is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredEmailVerificationCodes(ctx)
	user, err := appStore.VerifyUserEmailCode(ctx, email, hashOpaqueToken(code))
	if err != nil {
		if errors.Is(err, store.ErrVerificationCode) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "verification code is invalid or expired"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	if appMailer != nil {
		welcomeCtx, welcomeCancel := context.WithTimeout(context.Background(), 8*time.Second)
		if err := appMailer.SendWelcome(welcomeCtx, mailer.WelcomeEmailInput{
			To:           user.Email,
			Name:         user.Name,
			DashboardURL: buildDashboardURL(r, "alerts"),
		}); err != nil {
			log.Printf("welcome email failed for %s: %v", user.Email, err)
		}
		welcomeCancel()
	}

	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:            true,
		Authenticated: true,
		User:          toAuthUser(user),
		Message:       "Email verified. Access granted.",
		Email:         user.Email,
	})
}

func handleAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authForgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-resend-verification", email, rateLimitConfig{Limit: 3, Window: 10 * time.Minute}) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := appStore.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			writeJSON(w, http.StatusOK, authMessageResponse{
				OK:      true,
				Message: "If an account is pending verification, a new OTP has been issued.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if !user.VerificationRequired {
		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "This email is already verified. You can sign in now.",
			Email:   user.Email,
		})
		return
	}
	recent, err := appStore.HasRecentEmailVerificationCode(ctx, user.ID, time.Now().Add(-verifyResendCooldown))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if recent {
		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "A recent verification code is still active. Check your inbox before requesting another.",
			Email:   user.Email,
		})
		return
	}

	resp, err := prepareVerificationChallenge(ctx, r, user)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError{Error: err.Error()})
		return
	}
	resp.Message = "A fresh verification code has been sent."
	writeJSON(w, http.StatusOK, resp)
}

func handleAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authForgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-forgot-password", email, rateLimitConfig{Limit: 3, Window: 15 * time.Minute}) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredPasswordResetTokens(ctx)

	resp := authMessageResponse{
		OK:      true,
		Message: "If an account exists, a reset link has been issued.",
	}

	user, err := appStore.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			writeJSON(w, http.StatusOK, resp)
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	recent, err := appStore.HasRecentPasswordResetToken(ctx, user.ID, time.Now().Add(-resetIssueCooldown))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if recent {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	token, tokenHash, err := generatePasswordResetToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to create reset token"})
		return
	}
	expiresAt := time.Now().Add(defaultResetTTL)
	if err := appStore.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	resetURL := buildPasswordResetURL(r, token)
	if appMailer != nil {
		if err := appMailer.SendPasswordReset(ctx, mailer.PasswordResetInput{
			To:        user.Email,
			Name:      user.Name,
			ResetURL:  resetURL,
			ExpiresIn: defaultResetTTL,
		}); err != nil {
			if cleanupErr := appStore.DeletePasswordResetToken(ctx, user.ID); cleanupErr != nil {
				log.Printf("password reset token cleanup failed for %s: %v", user.Email, cleanupErr)
			}
			log.Printf("password reset email failed for %s: %v", user.Email, err)
			writeJSON(w, http.StatusBadGateway, apiError{Error: "email delivery is temporarily unavailable"})
			return
		}
		resp.Message = "If an account exists, a reset link has been sent to the inbox."
	} else {
		log.Printf("password reset requested for %s: %s (expires %s)", user.Email, resetURL, expiresAt.UTC().Format(time.RFC3339))
		resp.Message = "If an account exists, a reset link has been issued locally."
	}
	if appMailer == nil && isLocalRequestHost(r.Host) {
		resp.PreviewURL = resetURL
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "reset token is required"})
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if err := validatePasswordConfirmation(req.Password, req.ConfirmPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to hash password"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := appStore.DeleteExpiredPasswordResetTokens(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := appStore.ResetPasswordWithToken(ctx, hashOpaqueToken(token), string(passwordHash)); err != nil {
		if errors.Is(err, store.ErrResetTokenNotFound) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "reset link is invalid or expired"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	clearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:      true,
		Message: "Password updated. Sign in with your new password.",
	})
}

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

func currentSessionUser(r *http.Request) (store.User, string, error) {
	tokenHash, err := sessionTokenHashFromRequest(r)
	if err != nil {
		return store.User{}, "", err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := appStore.GetUserBySessionToken(ctx, tokenHash)
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

func getOrCreateDemoUser(ctx context.Context) (store.User, error) {
	user, err := appStore.GetUserByEmail(ctx, appDemoEmail)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, store.ErrUserNotFound) {
		return store.User{}, err
	}

	passwordToken, _, genErr := generateOpaqueToken()
	if genErr != nil {
		return store.User{}, fmt.Errorf("failed to generate demo credentials")
	}
	passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(passwordToken), bcrypt.DefaultCost)
	if hashErr != nil {
		return store.User{}, fmt.Errorf("failed to prepare demo account")
	}
	return appStore.CreateUser(ctx, appDemoName, appDemoEmail, string(passwordHash), false)
}

func isDemoUser(user store.User) bool {
	if !appDemoEnabled || appDemoEmail == "" {
		return false
	}
	return strings.EqualFold(user.Email, appDemoEmail)
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
		Name:     "netwatcher_demo",
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
		Name:     "netwatcher_preview",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Cf-Visitor")), "\"https\"") {
		return true
	}
	return false
}

func requestHost(r *http.Request) string {
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		return forwardedHost
	}
	return r.Host
}

func requestScheme(r *http.Request) string {
	if requestIsSecure(r) {
		return "https"
	}
	return "http"
}

func buildPasswordResetURL(r *http.Request, token string) string {
	if appPublicBaseURL != "" {
		return fmt.Sprintf("%s/reset-password?token=%s", appPublicBaseURL, url.QueryEscape(token))
	}
	return fmt.Sprintf("%s://%s/reset-password?token=%s", requestScheme(r), requestHost(r), url.QueryEscape(token))
}

func buildDashboardURL(r *http.Request, panel string) string {
	base := buildConfiguredDashboardURL(panel)
	if base != "" {
		return base
	}
	return fmt.Sprintf("%s://%s%s", requestScheme(r), requestHost(r), dashboardRoutePath(panel))
}

func buildConfiguredDashboardURL(panel string) string {
	base := strings.TrimRight(strings.TrimSpace(appPublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + dashboardRoutePath(panel)
}

func dashboardRoutePath(panel string) string {
	switch strings.TrimSpace(panel) {
	case "", "overview":
		return "/"
	case "ping":
		return "/icmp-ping"
	case "tcpping":
		return "/tcp-ping"
	case "portscan":
		return "/port-scan"
	case "dnslookup":
		return "/dns-lookup"
	case "history":
		return "/history"
	case "alerts":
		return "/alerts"
	case "settings":
		return "/settings"
	case "profile":
		return "/profile"
	default:
		return "/"
	}
}

func isLocalRequestHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.Contains(host, ":") {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func normalizeVerificationCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func prepareVerificationChallenge(ctx context.Context, r *http.Request, user store.User) (authMessageResponse, error) {
	code, codeHash, err := generateVerificationCode()
	if err != nil {
		return authMessageResponse{}, fmt.Errorf("failed to generate verification code")
	}
	expiresAt := time.Now().Add(defaultVerifyCodeTTL)
	if err := appStore.CreateEmailVerificationCode(ctx, user.ID, codeHash, expiresAt); err != nil {
		return authMessageResponse{}, err
	}

	resp := authMessageResponse{
		OK:                   true,
		Message:              "Account created. Enter the OTP sent to your email to activate access.",
		VerificationRequired: true,
		Email:                user.Email,
	}

	if appMailer != nil {
		if err := appMailer.SendVerificationOTP(ctx, mailer.VerificationOTPInput{
			To:        user.Email,
			Name:      user.Name,
			Code:      code,
			ExpiresIn: defaultVerifyCodeTTL,
		}); err != nil {
			if cleanupErr := appStore.DeleteEmailVerificationCode(ctx, user.ID); cleanupErr != nil {
				log.Printf("verification code cleanup failed for %s: %v", user.Email, cleanupErr)
			}
			log.Printf("verification email failed for %s: %v", user.Email, err)
			resp.Message = "Account created, but email delivery failed. Request a new OTP to continue."
			return resp, nil
		}
		return resp, nil
	}

	log.Printf("email verification requested for %s: otp=%s (expires %s)", user.Email, code, expiresAt.UTC().Format(time.RFC3339))
	resp.PreviewCode = code
	resp.Message = "Account created. Enter the OTP below to activate access."
	return resp, nil
}

func normalizeDisplayName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("name is required")
	}
	if len(value) > maxNameLength {
		return "", fmt.Errorf("name must be %d characters or fewer", maxNameLength)
	}
	return value, nil
}

func normalizeSignupDisplayName(name, firstName, lastName string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return normalizeDisplayName(name)
	}

	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	if firstName == "" {
		return "", fmt.Errorf("first name is required")
	}
	if lastName == "" {
		return "", fmt.Errorf("last name is required")
	}
	return normalizeDisplayName(firstName + " " + lastName)
}

func validatePassword(value string) error {
	if len(value) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	if len(value) > maxPasswordLength {
		return fmt.Errorf("password must be %d characters or fewer", maxPasswordLength)
	}
	return nil
}

func validatePasswordConfirmation(password, confirmPassword string) error {
	confirmPassword = strings.TrimSpace(confirmPassword)
	if confirmPassword == "" {
		return nil
	}
	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func toAuthUser(user store.User) *authUser {
	return &authUser{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		EmailVerifiedAt: user.EmailVerifiedAt,
		CreatedAt:       user.CreatedAt,
	}
}
