package app

import (
	"context"
	"log/slog"
	"time"
)

type App struct {
	log *slog.Logger
}

func New(logger *slog.Logger) *App {
	return &App{log: logger}
}

func (a *App) Start(ctx context.Context) error {
	a.log.Info("starting app")
	// init dependencies, DB connections, etc.
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.log.Info("stopping app", "timeout", "5s")
	// graceful shutdown
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
