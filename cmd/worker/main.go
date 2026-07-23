package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curefatih/afi/internal/adapters/eventpub"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/telemetry"
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

	tel, err := telemetry.Init(ctx, cfg, "afi-worker")
	if err != nil {
		log.Error("telemetry", "err", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			log.Error("telemetry shutdown", "err", err)
		}
	}()

	var workerMetrics *telemetry.WorkerMetrics
	if cfg.Telemetry.Enabled {
		workerMetrics, err = telemetry.NewWorkerMetrics()
		if err != nil {
			log.Error("worker metrics", "err", err)
			os.Exit(1)
		}
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	usageSrc := &postgres.UsageOutbox{Pool: pool}
	usageSink := &postgres.UsageSink{Pool: pool}
	prices := &postgres.PriceLookup{Pool: pool}

	var eventPub workers.EventPublisher
	var eventClose func()
	eventSrc := postgres.NewPlatformEventOutbox(pool)
	if cfg.Events.OutboxEnabled {
		eventPub, eventClose, err = eventpub.New(cfg, log)
		if err != nil {
			log.Error("event publisher", "err", err)
			os.Exit(1)
		}
		if eventClose != nil {
			defer eventClose()
		}
		log.Info("platform event drain enabled", "publisher", cfg.Events.Publisher)
	}

	log.Info("worker started", "poll", "2s", "telemetry", cfg.Telemetry.Enabled)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopped")
			return
		case <-ticker.C:
			n, err := workers.ProcessOnce(ctx, usageSrc, usageSink, prices, workerMetrics)
			if err != nil {
				log.Error("process usage", "err", err)
			} else if n > 0 {
				log.Info("processed usage outbox", "count", n)
			}

			if eventPub != nil {
				en, err := workers.ProcessPlatformEventsOnce(ctx, eventSrc, eventPub, workerMetrics)
				if err != nil {
					log.Error("process platform events", "err", err)
				} else if en > 0 {
					log.Info("processed platform event outbox", "count", en)
				}
			}
		}
	}
}
