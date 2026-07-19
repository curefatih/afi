package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/controlplane"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.2.0-dev"

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
	case "db":
		if len(os.Args) < 3 || os.Args[2] != "reset" {
			fmt.Fprintln(os.Stderr, "usage: afi db reset")
			os.Exit(2)
		}
		if err := runDBReset(); err != nil {
			fmt.Fprintf(os.Stderr, "db reset: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("database reset and migrated")
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
  afi db reset          # destructive; local only
`, version)
}

func open() (*kernel.Config, *pgxpool.Pool, context.Context, context.CancelFunc, error) {
	cfg, err := kernel.LoadConfig()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	snapStore := postgres.NewSnapshotStore(pool)
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
	snapStore := postgres.NewSnapshotStore(pool)
	seeder := controlplane.NewSeeder(pool, store, snapStore, cfg)
	return seeder.PublishSnapshot(ctx)
}

func runDBReset() error {
	fmt.Fprint(os.Stderr, "This DROPS all AFI tables. Type 'reset' to continue: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(line) != "reset" {
		return fmt.Errorf("aborted")
	}
	_, pool, ctx, cancel, err := open()
	if err != nil {
		return err
	}
	defer cancel()
	defer pool.Close()
	return controlplane.ResetDatabase(ctx, pool)
}
