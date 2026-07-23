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
	afiRedis "github.com/curefatih/afi/internal/adapters/redis"
	"github.com/curefatih/afi/internal/adapters/secrets"
	afiWasm "github.com/curefatih/afi/internal/adapters/wasm"
	"github.com/curefatih/afi/internal/credentials"
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

	var wasmMods []*afiWasm.Module
	defer func() {
		for _, m := range wasmMods {
			_ = m.Close(context.Background())
		}
	}()
	if path := cfg.Gateway.WasmBeforeCall; path != "" {
		mod, err := afiWasm.CompileFile(ctx, path, afiWasm.Config{Name: "wasm:before_call"})
		if err != nil {
			log.Error("wasm before_call", "path", path, "err", err)
			os.Exit(1)
		}
		hook, err := afiWasm.NewBeforeCall(mod)
		if err != nil {
			_ = mod.Close(ctx)
			log.Error("wasm before_call adapter", "err", err)
			os.Exit(1)
		}
		wasmMods = append(wasmMods, mod)
		hooks = hooks.RegisterBeforeCall(hook)
		log.Info("wasm before_call loaded", "path", path)
	}
	if path := cfg.Gateway.WasmBeforeChat; path != "" {
		mod, err := afiWasm.CompileFile(ctx, path, afiWasm.Config{Name: "wasm:before_chat"})
		if err != nil {
			log.Error("wasm before_chat", "path", path, "err", err)
			os.Exit(1)
		}
		hook, err := afiWasm.NewBeforeChat(mod)
		if err != nil {
			_ = mod.Close(ctx)
			log.Error("wasm before_chat adapter", "err", err)
			os.Exit(1)
		}
		wasmMods = append(wasmMods, mod)
		hooks = hooks.Register(hook)
		log.Info("wasm before_chat loaded", "path", path)
	}

	pipeline := dataplane.NewPipelineWithRegistry(holder, reg, log)
	pipeline.Hooks = hooks
	wasmCache := afiWasm.NewModuleCache(afiWasm.Config{Name: "snap-wasm"})
	defer func() { _ = wasmCache.Close(context.Background()) }()
	s3Fetcher, err := afiWasm.NewS3Fetcher(afiWasm.S3Config{
		Endpoint:  cfg.Gateway.WasmS3.Endpoint,
		AccessKey: cfg.Gateway.WasmS3.AccessKey,
		SecretKey: cfg.Gateway.WasmS3.SecretKey,
		Region:    cfg.Gateway.WasmS3.Region,
		UseSSL:    cfg.Gateway.WasmS3.UseSSL,
		PathStyle: cfg.Gateway.WasmS3.PathStyle,
	})
	if err != nil {
		log.Error("wasm s3", "err", err)
		os.Exit(1)
	}
	if s3Fetcher != nil {
		wasmCache.SetFetcher(s3Fetcher)
		log.Info("wasm s3 fetcher enabled", "endpoint", cfg.Gateway.WasmS3.Endpoint)
	}
	pipeline.Wasm = &dataplane.WasmRunner{Cache: wasmCache, Log: log}
	var credBox *credentials.Box
	if cfg.Credentials.MasterKey != "" {
		box, err := credentials.ParseMasterKey(cfg.Credentials.MasterKey)
		if err != nil {
			log.Error("credentials master key", "err", err)
			os.Exit(1)
		}
		credBox = box
	}
	vaultMulti := secrets.Multi{Env: secrets.Env{}}
	if cfg.Secrets.AWSSM.Enabled {
		awsSM, err := secrets.NewAWSSMFromEnv(ctx, true, cfg.Secrets.AWSSM.Region)
		if err != nil {
			log.Error("aws secrets manager", "err", err)
			os.Exit(1)
		}
		vaultMulti.AWSSM = awsSM
		log.Info("aws secrets manager enabled", "region", cfg.Secrets.AWSSM.Region)
	}
	if h := secrets.NewHashicorpFromEnv(cfg.Secrets.Vault.Addr, cfg.Secrets.Vault.Token); h != nil {
		vaultMulti.Hashicorp = h
		log.Info("hashicorp vault enabled", "addr", h.Addr)
	}
	pipeline.Credentials = secrets.NewCredentialResolver(credBox).WithVault(vaultMulti)
	pipeline.Secrets = vaultMulti

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
		timed = &afiRedis.Counters{Client: rdb}
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
		payload, err := workers.EncodeUsage(e)
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
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", "err", err)
	}
	log.Info("stopped")
}
