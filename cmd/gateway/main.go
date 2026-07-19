package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curefatih/afi/extensions/demohook"
	"github.com/curefatih/afi/extensions/echo"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/dataplane"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log := kernel.NewLogger("gateway")

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

	snapStore := postgres.NewSnapshotStore(pool)
	holder := dataplane.NewHolder()
	reg := dataplane.DefaultRegistry().RegisterSDK(echo.New())
	hooks := dataplane.NewHookChain().RegisterHook(demohook.NewWithLog(log))
	pipeline := dataplane.NewPipelineWithRegistry(holder, reg, log)
	pipeline.Hooks = hooks

	var timed dataplane.CounterStore
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Error("redis url", "err", err)
			os.Exit(1)
		}
		rdb := redis.NewClient(opt)
		defer rdb.Close()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Warn("redis unavailable; timed quota windows will fail until Redis is up", "err", err)
		} else {
			log.Info("redis connected", "addr", opt.Addr)
		}
		timed = &dataplane.RedisCounters{Client: rdb}
	}
	pipeline.Counters = dataplane.CompositeCounters{
		Total: &postgres.Counters{Pool: pool},
		Timed: timed,
	}

	polEval, err := policy.NewEvaluator()
	if err != nil {
		log.Error("policy evaluator", "err", err)
		os.Exit(1)
	}
	pipeline.Policies = polEval
	log.Info("extensions registered", "provider_types", reg.Types(), "hooks", hooks.Infos())

	outbox := &postgres.UsageOutbox{Pool: pool}
	pipeline.Usage = func(e dataplane.UsageEvent) {
		payload, err := workers.EncodeUsage(workers.UsagePayload{
			OrganizationID:   e.OrganizationID,
			ProjectID:        e.ProjectID,
			APIKeyID:         e.APIKeyID,
			Model:            e.Model,
			ProviderType:     e.ProviderType,
			TargetModel:      e.TargetModel,
			Status:           e.Status,
			LatencyMs:        e.LatencyMs,
			PromptTokens:     e.PromptTokens,
			CompletionTokens: e.CompletionTokens,
			Modality:         e.Modality,
			Metrics:          e.Metrics,
		})
		if err != nil {
			log.Error("encode usage", "err", err)
			return
		}
		if err := outbox.Enqueue(context.Background(), payload); err != nil {
			log.Error("enqueue usage", "err", err)
		}
	}

	go func() {
		err := snapStore.Watch(ctx, cfg.Gateway.SnapshotPollInterval, func(s *snapshot.Snapshot) {
			holder.Set(s)
			log.Info("snapshot loaded", "version", s.Version, "keys", len(s.APIKeys), "routes", len(s.Routes), "quotas", len(s.Quotas), "policies", len(s.Policies))
		})
		if err != nil && ctx.Err() == nil {
			log.Error("snapshot watch", "err", err)
			cancel()
		}
	}()

	httpServer := &http.Server{
		Addr:              cfg.Gateway.Addr,
		Handler:           pipeline.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.Gateway.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Info("stopped")
}
