package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/vsamtuc/mcm/pkg/auth"
)

// Config controls JWT middleware behavior.
type Config struct {
	IssuerURL        string
	ClientID         string
	SkipPaths        []string
	HTTPTimeout      time.Duration
	SessionCookie    string
	BrowserIssuerURL string
}

// Middleware validates bearer tokens issued by Keycloak and injects the user into the request context.
type Middleware struct {
	verifier       *oidc.IDTokenVerifier
	skipPaths      map[string]struct{}
	cookieName     string
	allowedIssuers []string
}

// New constructs the middleware using discovery from the issuer URL.
func New(ctx context.Context, cfg Config) (*Middleware, error) {
	if cfg.IssuerURL == "" {
		return nil, errors.New("issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("client ID is required")
	}
	ctx, cancel := context.WithTimeout(ctx, timeoutOrDefault(cfg.HTTPTimeout))
	defer cancel()
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID, SkipIssuerCheck: true})
	allowed := uniqueIssuers([]string{normalizeIssuer(cfg.IssuerURL), normalizeIssuer(cfg.BrowserIssuerURL)})
	skips := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		if p == "" {
			continue
		}
		skips[p] = struct{}{}
	}
	cookie := strings.TrimSpace(cfg.SessionCookie)
	return &Middleware{verifier: verifier, skipPaths: skips, cookieName: cookie, allowedIssuers: allowed}, nil
}

// Wrap applies the middleware to an http.Handler.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.tokenFromRequest(r)
		if err != nil {
			if errors.Is(err, errMissingToken) && m.shouldSkip(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		idToken, err := m.verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid token: %v", err), http.StatusUnauthorized)
			return
		}
		if !m.isIssuerAllowed(idToken.Issuer) {
			http.Error(w, fmt.Sprintf("invalid issuer: %s", idToken.Issuer), http.StatusUnauthorized)
			return
		}

		claims, err := extractClaims(idToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid claims: %v", err), http.StatusUnauthorized)
			return
		}

		ctx := auth.WithUser(r.Context(), auth.User{
			Subject:  idToken.Subject,
			Username: claims.PreferredUsername,
			Email:    claims.Email,
			Roles:    claims.Roles,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var errMissingToken = errors.New("missing token")

func (m *Middleware) tokenFromRequest(r *http.Request) (string, error) {
	if token, err := bearerToken(r.Header.Get("Authorization")); err == nil {
		return token, nil
	}
	if m.cookieName != "" && r != nil {
		if c, err := r.Cookie(m.cookieName); err == nil {
			val := strings.TrimSpace(c.Value)
			if val != "" {
				return val, nil
			}
		}
	}
	return "", errMissingToken
}

func (m *Middleware) shouldSkip(path string) bool {
	if path == "" {
		return false
	}
	_, ok := m.skipPaths[path]
	return ok
}

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing Authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("invalid Authorization header")
	}
	return strings.TrimSpace(parts[1]), nil
}

type keycloakClaims struct {
	PreferredUsername string                  `json:"preferred_username"`
	Email             string                  `json:"email"`
	ResourceAccess    map[string]clientAccess `json:"resource_access"`
	RealmAccess       clientAccess            `json:"realm_access"`
	Roles             []string                `json:"-"`
}

type clientAccess struct {
	Roles []string `json:"roles"`
}

func extractClaims(idToken *oidc.IDToken) (*keycloakClaims, error) {
	var claims keycloakClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	roleSet := map[string]struct{}{}
	for _, r := range claims.RealmAccess.Roles {
		roleSet[r] = struct{}{}
	}
	for _, access := range claims.ResourceAccess {
		for _, r := range access.Roles {
			roleSet[r] = struct{}{}
		}
	}
	claims.Roles = make([]string, 0, len(roleSet))
	for role := range roleSet {
		claims.Roles = append(claims.Roles, role)
	}
	return &claims, nil
}

func timeoutOrDefault(d time.Duration) time.Duration {
	if d <= 0 {
		return 5 * time.Second
	}
	return d
}

func (m *Middleware) isIssuerAllowed(issuer string) bool {
	norm := normalizeIssuer(issuer)
	for _, allowed := range m.allowedIssuers {
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
