package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	app "github.com/vsamtuc/mcm/internal/app"
	devauth "github.com/vsamtuc/mcm/internal/auth/devel"
	authjwt "github.com/vsamtuc/mcm/internal/auth/jwt"
	httpx "github.com/vsamtuc/mcm/internal/transport/http"
	"github.com/vsamtuc/mcm/pkg/auth"
)

func main() {

	develFlag := parseFlags()
	if develFlag {
		_ = os.Setenv("MCM_DEVEL_MODE", "1")
	} else {
		_ = os.Setenv("MCM_DEVEL_MODE", "0")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a := app.New(logger)
	devel := a.DevelMode()

	if err := a.Start(context.Background()); err != nil {
		logger.Error("app start failed", "err", err)
		os.Exit(1)
	}

	authCfg := httpx.AuthConfig{SessionCookie: httpx.SessionCookieName, DevMode: devel}
	var issuerURL, browserIssuerURL, clientID string
	var err error
	if !devel {
		issuerURL, err = resolveIssuerURL()
		if err != nil {
			logger.Error("failed to resolve issuer", "err", err)
			os.Exit(1)
		}
		browserIssuerURL, err = resolveBrowserIssuerURL()
		if err != nil {
			logger.Error("failed to resolve browser issuer", "err", err)
			os.Exit(1)
		}
		if browserIssuerURL == "" {
			browserIssuerURL = issuerURL
		}
		clientID = strings.TrimSpace(os.Getenv("KEYCLOAK_CLIENT_ID"))
		if clientID == "" {
			logger.Error("KEYCLOAK_CLIENT_ID must be set")
			os.Exit(1)
		}
		authCfg.IssuerURL = issuerURL
		authCfg.BrowserURL = browserIssuerURL
		authCfg.ClientID = clientID
	}
	handler, err := httpx.NewMux(a.SchemaReady, a.Service(), authCfg)
	if err != nil {
		logger.Error("failed to init http mux", "err", err)
		os.Exit(1)
	}
	var authHandler http.Handler
	if devel {
		devMW := devauth.New(devauth.Config{
			CookieName:  httpx.SessionCookieName,
			DefaultUser: auth.User{Subject: "dev-admin", Username: "Dev Admin", Email: "dev-admin@example.com", Roles: []string{"admin", "professor"}},
			SkipPaths:   []string{"/livez", "/readyz"},
		})
		authHandler = devMW.Wrap(handler)
	} else {
		authHandler, err = withAuthMiddleware(handler, issuerURL, browserIssuerURL, clientID)
		if err != nil {
			logger.Error("failed to init auth middleware", "err", err)
			os.Exit(1)
		}
	}

	logger.Info("auth middleware initialized")
	loggedHandler := logRequests(logger, authHandler)

	srv := &http.Server{Addr: ":8080", Handler: loggedHandler}

	go func() {
		logger.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
	_ = a.Stop(ctx)
}

func withAuthMiddleware(next http.Handler, issuerURL, browserIssuerURL, clientID string) (http.Handler, error) {
	skipPaths := []string{"/", "/courses", httpx.LoginPath, httpx.LogoutPath, httpx.CallbackPath, "/livez", "/readyz"}
	skipPaths = append(skipPaths, additionalSkipPaths()...)
	mw, err := authjwt.New(context.Background(), authjwt.Config{
		IssuerURL:        issuerURL,
		BrowserIssuerURL: browserIssuerURL,
		ClientID:         clientID,
		SkipPaths:        skipPaths,
		SessionCookie:    httpx.SessionCookieName,
	})
	if err != nil {
		return nil, err
	}
	return mw.Wrap(next), nil
}

func resolveIssuerURL() (string, error) {
	if issuer := strings.TrimSpace(os.Getenv("KEYCLOAK_ISSUER_URL")); issuer != "" {
		return issuer, nil
	}
	baseURL := strings.TrimSpace(os.Getenv("KEYCLOAK_URL"))
	if baseURL == "" {
		return "", fmt.Errorf("KEYCLOAK_URL or KEYCLOAK_ISSUER_URL must be set")
	}
	realm := strings.TrimSpace(os.Getenv("KEYCLOAK_REALM"))
	if realm == "" {
		realm = "mcm"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/realms/%s", baseURL, realm), nil
}

func resolveBrowserIssuerURL() (string, error) {
	if issuer := strings.TrimSpace(os.Getenv("KEYCLOAK_BROWSER_ISSUER_URL")); issuer != "" {
		return issuer, nil
	}
	baseURL := strings.TrimSpace(os.Getenv("KEYCLOAK_BROWSER_URL"))
	if baseURL == "" {
		return "", nil
	}
	realm := strings.TrimSpace(os.Getenv("KEYCLOAK_REALM"))
	if realm == "" {
		realm = "mcm"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/realms/%s", baseURL, realm), nil
}

func additionalSkipPaths() []string {
	configured := strings.TrimSpace(os.Getenv("AUTH_SKIP_PATHS"))
	if configured == "" {
		return nil
	}
	parts := strings.Split(configured, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		duration := time.Since(start)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", duration,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func parseFlags() bool {
	defaultDevel := envDevelEnabled()
	devel := flag.Bool("devel", defaultDevel, "enable development mode (equivalent to setting MCM_DEVEL_MODE)")
	flag.Parse()
	return *devel
}

func envDevelEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("MCM_DEVEL_MODE")))
	return val == "1" || val == "true" || val == "yes" || val == "on"
}
