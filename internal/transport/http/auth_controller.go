package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	// SessionCookieName stores the ID token returned by Keycloak.
	SessionCookieName = "mcm_session"
	// LoginPath starts the OAuth2 authorization code flow.
	LoginPath = "/auth/login"
	// CallbackPath receives the authorization code exchange result.
	CallbackPath = "/auth/callback"
	// LogoutPath clears the local session and redirects to Keycloak logout.
	LogoutPath = "/auth/logout"

	stateCookieName = "mcm_oauth_state"
	pkceCookieName  = "mcm_oauth_pkce"
)

// AuthConfig controls the HTML auth surface.
type AuthConfig struct {
	IssuerURL     string
	ClientID      string
	BrowserURL    string
	SessionCookie string
	StateCookie   string
	PKCECookie    string
	// Timeout controls OAuth calls; zero uses the default.
	Timeout time.Duration
}

type authController struct {
	oauthConfig    oauth2.Config
	verifier       *oidc.IDTokenVerifier
	issuer         string
	browserIssuer  string
	clientID       string
	sessionCookie  string
	stateCookie    string
	pkceCookie     string
	httpTimeout    time.Duration
	allowedIssuers []string
}

func newAuthController(ctx context.Context, cfg AuthConfig) (*authController, error) {
	if cfg.IssuerURL == "" {
		return nil, errors.New("issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	oauthCfg := oauth2.Config{
		ClientID: cfg.ClientID,
		Endpoint: provider.Endpoint(),
		Scopes:   []string{oidc.ScopeOpenID, "profile", "email"},
	}

	issuer := normalizeIssuer(cfg.IssuerURL)
	browserIssuer := normalizeIssuer(cfg.BrowserURL)
	if browserIssuer == "" {
		browserIssuer = issuer
	}
	oauthCfg.Endpoint.AuthURL = fmt.Sprintf("%s/protocol/openid-connect/auth", browserIssuer)
	allowedIssuers := uniqueIssuers([]string{issuer, browserIssuer})

	sessionCookie := cfg.SessionCookie
	if sessionCookie == "" {
		sessionCookie = SessionCookieName
	}
	stateCookie := cfg.StateCookie
	if stateCookie == "" {
		stateCookie = stateCookieName
	}
	pkceCookie := cfg.PKCECookie
	if pkceCookie == "" {
		pkceCookie = pkceCookieName
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &authController{
		oauthConfig:    oauthCfg,
		verifier:       provider.Verifier(&oidc.Config{ClientID: cfg.ClientID, SkipIssuerCheck: true}),
		issuer:         issuer,
		browserIssuer:  browserIssuer,
		clientID:       cfg.ClientID,
		sessionCookie:  sessionCookie,
		stateCookie:    stateCookie,
		pkceCookie:     pkceCookie,
		httpTimeout:    timeout,
		allowedIssuers: allowedIssuers,
	}, nil
}

func (a *authController) register(mux *http.ServeMux) {
	mux.HandleFunc(LoginPath, a.login)
	mux.HandleFunc(CallbackPath, a.callback)
	mux.HandleFunc(LogoutPath, a.logout)
}

func (a *authController) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := randomURLSafeString(32)
	if err != nil {
		http.Error(w, "generate state", http.StatusInternalServerError)
		return
	}
	codeVerifier, err := randomURLSafeString(64)
	if err != nil {
		http.Error(w, "generate verifier", http.StatusInternalServerError)
		return
	}

	a.setTransientCookie(w, r, a.stateCookie, state, 10*time.Minute)
	a.setTransientCookie(w, r, a.pkceCookie, codeVerifier, 10*time.Minute)

	cfg := a.configWithRedirect(a.callbackURL(r))
	authURL := cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (a *authController) callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if errStr := r.URL.Query().Get("error"); errStr != "" {
		desc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("authorization error: %s %s", errStr, desc), http.StatusBadRequest)
		return
	}

	expectedState, err := r.Cookie(a.stateCookie)
	if err != nil {
		http.Error(w, "state cookie missing", http.StatusBadRequest)
		return
	}
	codeVerifierCookie, err := r.Cookie(a.pkceCookie)
	if err != nil {
		http.Error(w, "pkce verifier missing", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != expectedState.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "authorization code missing", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), a.httpTimeout)
	defer cancel()

	cfg := a.configWithRedirect(a.callbackURL(r))
	token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifierCookie.Value))
	if err != nil {
		http.Error(w, fmt.Sprintf("exchange code: %v", err), http.StatusBadRequest)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "id token missing", http.StatusBadRequest)
		return
	}

	idToken, err := a.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("verify id token: %v", err), http.StatusBadRequest)
		return
	}
	if !a.isIssuerAllowed(idToken.Issuer) {
		http.Error(w, fmt.Sprintf("unexpected issuer: %s", idToken.Issuer), http.StatusBadRequest)
		return
	}

	a.clearCookie(w, a.stateCookie)
	a.clearCookie(w, a.pkceCookie)
	a.setSessionCookie(w, r, rawIDToken, time.Until(idToken.Expiry))

	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *authController) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.clearCookie(w, a.sessionCookie)
	a.clearCookie(w, a.stateCookie)
	a.clearCookie(w, a.pkceCookie)

	redirectURI := absoluteURL(r, "/")
	values := url.Values{}
	values.Set("client_id", a.clientID)
	values.Set("post_logout_redirect_uri", redirectURI)

	if idCookie, err := r.Cookie(a.sessionCookie); err == nil && idCookie.Value != "" {
		values.Set("id_token_hint", idCookie.Value)
	}

	logoutURL := fmt.Sprintf("%s/protocol/openid-connect/logout?%s", a.browserIssuer, values.Encode())
	http.Redirect(w, r, logoutURL, http.StatusFound)
}

func (a *authController) configWithRedirect(redirect string) oauth2.Config {
	cfg := a.oauthConfig
	cfg.RedirectURL = redirect
	return cfg
}

func (a *authController) callbackURL(r *http.Request) string {
	return absoluteURL(r, CallbackPath)
}

func (a *authController) setTransientCookie(w http.ResponseWriter, r *http.Request, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

func (a *authController) setSessionCookie(w http.ResponseWriter, r *http.Request, value string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	http.SetCookie(w, &http.Cookie{
		Name:     a.sessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

func (a *authController) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:   name,
		Path:   "/",
		MaxAge: -1,
	})
}

func randomURLSafeString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func isSecureRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if proto := strings.ToLower(r.Header.Get("X-Forwarded-Proto")); proto == "https" {
		return true
	}
	return r.TLS != nil
}

func absoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}
	if host == "" {
		host = "localhost:8080"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func (a *authController) isIssuerAllowed(issuer string) bool {
	norm := normalizeIssuer(issuer)
	for _, allowed := range a.allowedIssuers {
		if allowed == norm {
			return true
		}
	}
	return false
}

func normalizeIssuer(issuer string) string {
	issuer = strings.TrimSpace(issuer)
	issuer = strings.TrimSuffix(issuer, "/")
	return issuer
}

func uniqueIssuers(issuers []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(issuers))
	for _, iss := range issuers {
		if iss == "" {
			continue
		}
		if _, ok := seen[iss]; ok {
			continue
		}
		seen[iss] = struct{}{}
		result = append(result, iss)
	}
	return result
}
