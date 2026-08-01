package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// version is overridden at link time by scripts/build-release.sh (-X main.version=...).
var version = "0.3.0-dev"

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
	case "regions":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: afi regions bind-all <region-slug-or-id>")
			os.Exit(2)
		}
		switch os.Args[2] {
		case "bind-all":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: afi regions bind-all <region-slug-or-id>")
				os.Exit(2)
			}
			n, err := runBindAll(os.Args[3])
			if err != nil {
				fmt.Fprintf(os.Stderr, "regions bind-all: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("bound %d organization(s); snapshot published\n", n)
		default:
			fmt.Fprintln(os.Stderr, "usage: afi regions bind-all <region-slug-or-id>")
			os.Exit(2)
		}
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
  afi regions bind-all <region-slug-or-id>
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

	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}
	store := postgres.NewStore(pool)
	snapStore := postgres.NewSnapshotStore(pool)
	seeder := postgres.NewSeeder(pool, store, snapStore, cfg)
	return seeder.Seed(ctx)
}

func runPublish() error {
	cfg, pool, ctx, cancel, err := open()
	if err != nil {
		return err
	}
	defer cancel()
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}
	store := postgres.NewStore(pool)
	snapStore := postgres.NewSnapshotStore(pool)
	seeder := postgres.NewSeeder(pool, store, snapStore, cfg)
	return seeder.PublishSnapshot(ctx)
}

func runBindAll(slugOrID string) (int, error) {
	cfg, pool, ctx, cancel, err := open()
	if err != nil {
		return 0, err
	}
	defer cancel()
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		return 0, err
	}
	store := postgres.NewStore(pool)
	reg, err := store.GetRegion(ctx, slugOrID)
	if err != nil {
		reg, err = store.GetRegionBySlug(ctx, slugOrID)
		if err != nil {
			return 0, err
		}
	}
	n, err := store.BindAllOrgsToRegion(ctx, reg.ID)
	if err != nil {
		return n, err
	}
	snapStore := postgres.NewSnapshotStore(pool)
	seeder := postgres.NewSeeder(pool, store, snapStore, cfg)
	if err := seeder.PublishRegionSnapshots(ctx, reg.ID); err != nil {
		return n, err
	}
	return n, nil
}

func runDBReset() error {
	fmt.Fprint(os.Stderr, "This DROPS all AFI tables. Type 'reset' to continue: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("aborted: could not read confirmation: %w", err)
	}
	if strings.TrimSpace(line) != "reset" {
		return fmt.Errorf("aborted")
	}
	_, pool, ctx, cancel, err := open()
	if err != nil {
		return err
	}
	defer cancel()
	defer pool.Close()
	return postgres.ResetDatabase(ctx, pool)
}
