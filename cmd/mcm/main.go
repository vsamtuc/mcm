package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	app "github.com/vsamtuc/mcm/internal/app"
	httpx "github.com/vsamtuc/mcm/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a := app.New(logger)

	if err := a.Start(context.Background()); err != nil {
		logger.Error("app start failed", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{Addr: ":8080", Handler: httpx.NewMux()}

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
