package postgres

import (
	"context"
	"time"

	"github.com/curefatih/afi/internal/access"
	adapterauth "github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Seeder struct {
	store     *Store
	snapStore snapshot.Store
	cfg       *kernel.Config
	seed      *SeedWriter
}

func NewSeeder(pool *pgxpool.Pool, store *Store, snapStore snapshot.Store, cfg *kernel.Config) *Seeder {
	return &Seeder{
		store:     store,
		snapStore: snapStore,
		cfg:       cfg,
		seed:      NewSeedWriter(pool),
	}
}

// SeedIfEmpty inserts local-dev data when the database has no organizations.
// When the DB already has orgs, it still ensures local audio + echo extension routes (idempotent).
func (s *Seeder) SeedIfEmpty(ctx context.Context) error {
	n, err := s.store.CountOrgs(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return s.Seed(ctx)
	}
	if err := s.EnsureLocalAudioRoutes(ctx); err != nil {
		return err
	}
	return s.EnsureEchoExtension(ctx)
}

// EnsureEchoExtension upserts prov_echo + echo-demo route for org_local and republishes.
func (s *Seeder) EnsureEchoExtension(ctx context.Context) error {
	orgID := "org_local"
	exists, err := s.seed.OrgExists(ctx, orgID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := s.seed.UpsertEchoExtension(ctx, orgID, time.Now().UTC()); err != nil {
		return err
	}
	return s.PublishSnapshot(ctx)
}

// EnsureLocalAudioRoutes upserts tts-1 / whisper-1 → prov_openai for org_local and republishes.
func (s *Seeder) EnsureLocalAudioRoutes(ctx context.Context) error {
	orgID := "org_local"
	providerID := "prov_openai"
	exists, err := s.seed.OrgExists(ctx, orgID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	exists, err = s.seed.ProviderExists(ctx, providerID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := s.seed.EnsureAudioRoutes(ctx, orgID, providerID, time.Now().UTC()); err != nil {
		return err
	}
	// Always republish so capability normalize (tts/stt) reaches the gateway after upgrades.
	return s.PublishSnapshot(ctx)
}

// Seed always inserts (or upserts) the standard local-dev dataset and publishes a snapshot.
func (s *Seeder) Seed(ctx context.Context) error {
	cfg := s.cfg.Seed
	hash, err := adapterauth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	err = s.seed.SeedLocalDev(ctx, LocalDevSeed{
		OrgID:           "org_local",
		TeamID:          "team_local",
		ProjectID:       "proj_local",
		ProviderID:      "prov_openai",
		UserID:          "user_admin",
		RouteID:         "route_default",
		KeyID:           "key_local",
		AdminEmail:      cfg.AdminEmail,
		AdminName:       cfg.AdminName,
		PasswordHash:    hash,
		OpenAIBaseURL:   cfg.OpenAIBaseURL,
		OpenAIAPIKeyEnv: cfg.OpenAIAPIKeyEnv,
		DefaultModel:    cfg.DefaultModel,
		APIKeyHash:      access.Hash(cfg.VirtualAPIKey),
		APIKeyPrefix:    access.Prefix(cfg.VirtualAPIKey),
		Now:             time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	return s.PublishSnapshot(ctx)
}

func (s *Seeder) PublishSnapshot(ctx context.Context) error {
	src, err := s.store.LoadSnapshotSource(ctx)
	if err != nil {
		return err
	}
	snap := snapshot.Compile(src)
	_, err = s.snapStore.Put(ctx, snap)
	return err
}
