package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/controlplane"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := kernel.NewLogger("controlplane")

	cfg, err := kernel.LoadConfig()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := controlplane.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	store := controlplane.NewStore(pool)
	if err := store.SetCredentialsMasterKey(cfg.Credentials.MasterKey); err != nil {
		log.Error("credentials master key", "err", err)
		os.Exit(1)
	}
	snapStore := postgres.NewSnapshotStore(pool)
	seeder := controlplane.NewSeeder(pool, store, snapStore, cfg)

	if err := seeder.SeedIfEmpty(ctx); err != nil {
		log.Error("seed", "err", err)
		os.Exit(1)
	}

	// Ensure a snapshot exists even if data was seeded earlier without publish.
	if _, err := snapStore.Latest(ctx); err != nil {
		if err := seeder.PublishSnapshot(ctx); err != nil {
			log.Error("publish snapshot", "err", err)
			os.Exit(1)
		}
	}

	var eventOutbox platform.EventEnqueuer
	if cfg.Events.OutboxEnabled {
		eventOutbox = postgres.NewPlatformEventOutbox(pool)
		log.Info("platform event outbox enabled")
	}

	srv := controlplane.NewServer(cfg, store, seeder, snapStore, log, eventOutbox)
	httpServer := &http.Server{
		Addr:              cfg.ControlPlane.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.ControlPlane.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", "err", err)
	}
	log.Info("stopped")
}
