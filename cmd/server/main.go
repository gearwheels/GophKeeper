// Package main — точка входа сервера GophKeeper.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/timofeevav/gophkeeper/internal/server/config"
	"github.com/timofeevav/gophkeeper/internal/server/handler"
	"github.com/timofeevav/gophkeeper/internal/server/middleware"
	"github.com/timofeevav/gophkeeper/internal/server/migration"
	"github.com/timofeevav/gophkeeper/internal/server/repository/postgres"
	"github.com/timofeevav/gophkeeper/internal/server/service"
	"github.com/timofeevav/gophkeeper/internal/server/tlsconf"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Сигнал завершения приходит через контекст — его удобно прокидывать
	// дальше по цепочке инициализации компонентов.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	setupLogger(cfg.LogLevel)

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(connectCtx, cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migration.Run(pool); err != nil {
		return err
	}

	tlsCfg, err := tlsconf.Load(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return err
	}

	userRepo := postgres.NewUserRepo(pool)
	secretRepo := postgres.NewSecretRepo(pool)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	secretSvc := service.NewSecretService(secretRepo)
	syncSvc := service.NewSyncService(secretRepo)

	authH := handler.NewAuthHandler(authSvc)
	secretH := handler.NewSecretHandler(secretSvc)
	syncH := handler.NewSyncHandler(syncSvc)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))

			r.Post("/secrets", secretH.Create)
			r.Get("/secrets", secretH.List)
			r.Get("/secrets/{id}", secretH.Get)
			r.Put("/secrets/{id}", secretH.Update)
			r.Delete("/secrets/{id}", secretH.Delete)

			r.Post("/sync", syncH.Sync)
		})
	})

	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", cfg.ServerAddress, "tls", true)
		// Сертификат и ключ уже в TLSConfig — пути не нужны.
		if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return nil
}

func setupLogger(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
