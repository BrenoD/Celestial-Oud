// Celestial Oud API — single static binary, runs in a Cloudflare Container
// (or anywhere a container runs) in front of Supabase Postgres.
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

	"github.com/BrenoD/Celestial-Oud/backend/internal/api"
	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/config"
	"github.com/BrenoD/Celestial-Oud/backend/internal/db"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/mail"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer database.Close()
	if os.Getenv("SKIP_MIGRATIONS") != "1" {
		if err := database.Migrate(ctx, log); err != nil {
			log.Error("migrate", "err", err)
			os.Exit(1)
		}
	}

	st := store.New(database.Pool)
	httpx.TrustProxy = cfg.TrustProxy
	authSvc := auth.New(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc.RoleOf = st.UserRole
	mailer := mail.New(cfg.ResendAPIKey, cfg.EmailFrom, cfg.AdminEmails, cfg.PublicURL, st, log)
	handler := api.New(cfg, st, authSvc, mailer, log).Router()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("listening", "port", cfg.Port, "env", cfg.Env, "payments", cfg.PaymentsMode, "trust_proxy", cfg.TrustProxy)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("stopped")
}
