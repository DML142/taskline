package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskline/apps/api/internal/auth"
	"taskline/apps/api/internal/invite"
	"taskline/apps/api/internal/issue"
	"taskline/apps/api/internal/platform/config"
	"taskline/apps/api/internal/platform/database"
	httpserver "taskline/apps/api/internal/platform/http"
	"taskline/apps/api/internal/project"
	"taskline/apps/api/internal/workspace"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	authService := auth.NewService(auth.NewRepository(pool), auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience, time.Now), time.Now)
	authentication := auth.NewHandler(logger, authService, cfg.Auth)
	workspaces := workspace.NewHandler(workspace.NewService(workspace.NewRepository(pool)), authService)
	workspaceService := workspace.NewService(workspace.NewRepository(pool))
	invites := invite.NewHandler(invite.NewService(invite.NewRepository(pool), workspaceService, invite.NewSMTPMailer(cfg.SMTP), cfg.Auth.WebOrigin, time.Now), authService)
	projects := project.NewHandler(project.NewService(project.NewRepository(pool)), authService)
	issues := issue.NewHandler(issue.NewService(issue.NewRepository(pool)), authService)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.NewRouter(logger, pool, authentication.Routes(), workspaces, projects, issues, invites),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("api starting", "address", cfg.HTTPAddr)
		serveErr <- server.ListenAndServe()
	}()
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return errors.Join(fmt.Errorf("shutdown HTTP: %w", err), server.Close())
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("api stopped")
	return nil
}
