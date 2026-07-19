package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/controlplane"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := kernel.NewLogger("worker")

	cfg, err := kernel.LoadConfig()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := controlplane.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	src := &postgres.UsageOutbox{Pool: pool}
	sink := &postgres.UsageSink{Pool: pool}
	prices := &postgres.PriceLookup{Pool: pool}

	log.Info("worker started", "poll", "2s")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopped")
			return
		case <-ticker.C:
			n, err := workers.ProcessOnce(ctx, src, sink, prices)
			if err != nil {
				log.Error("process", "err", err)
				continue
			}
			if n > 0 {
				log.Info("processed usage outbox", "count", n)
			}
		}
	}
}
