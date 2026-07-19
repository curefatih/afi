package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/curefatih/afi/internal/controlplane"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "seed":
		if err := runSeed(); err != nil {
			fmt.Fprintf(os.Stderr, "seed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("seeded and snapshot published")
	case "snapshot":
		if len(os.Args) < 3 || os.Args[2] != "publish" {
			fmt.Fprintln(os.Stderr, "usage: afi snapshot publish")
			os.Exit(2)
		}
		if err := runPublish(); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot publish: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("snapshot published")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `afi %s

Usage:
  afi version
  afi seed
  afi snapshot publish
`, version)
}

func open() (*kernel.Config, *pgxpool.Pool, context.Context, context.CancelFunc, error) {
	cfg, err := kernel.LoadConfig()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		cancel()
		return nil, nil, nil, nil, err
	}
	return cfg, pool, ctx, cancel, nil
}

func runSeed() error {
	cfg, pool, ctx, cancel, err := open()
	if err != nil {
		return err
	}
	defer cancel()
	defer pool.Close()

	if err := controlplane.Migrate(ctx, pool); err != nil {
		return err
	}
	store := controlplane.NewStore(pool)
	snapStore := snapshot.NewStore(pool)
	seeder := controlplane.NewSeeder(pool, store, snapStore, cfg)
	return seeder.Seed(ctx)
}

func runPublish() error {
	cfg, pool, ctx, cancel, err := open()
	if err != nil {
		return err
	}
	defer cancel()
	defer pool.Close()

	if err := controlplane.Migrate(ctx, pool); err != nil {
		return err
	}
	store := controlplane.NewStore(pool)
	snapStore := snapshot.NewStore(pool)
	seeder := controlplane.NewSeeder(pool, store, snapStore, cfg)
	return seeder.PublishSnapshot(ctx)
}
