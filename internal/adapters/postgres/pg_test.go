package postgres_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/controlplane"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDSN = "postgres://afi:afi@localhost:5433/afi_test?sslmode=disable"

var (
	pgOnce sync.Once
	pgPool *pgxpool.Pool
	pgErr  error
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pgOnce.Do(func() {
		dsn := strings.TrimSpace(os.Getenv("AFI_TEST_DATABASE_URL"))
		if dsn == "" {
			dsn = defaultTestDSN
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ensureTestDatabase(ctx, dsn); err != nil {
			pgErr = err
			return
		}
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			pgErr = err
			return
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			pgErr = err
			return
		}
		if err := controlplane.ResetDatabase(ctx, pool); err != nil {
			pool.Close()
			pgErr = err
			return
		}
		pgPool = pool
	})
	if pgErr != nil {
		t.Skipf("postgres integration tests skipped: %v (set AFI_TEST_DATABASE_URL)", pgErr)
	}
	if pgPool == nil {
		t.Skip("postgres integration tests skipped")
	}
	return pgPool
}

func ensureTestDatabase(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err == nil {
		pingErr := pool.Ping(ctx)
		pool.Close()
		if pingErr == nil {
			return nil
		}
		err = pingErr
	}
	// Database may not exist yet — create it via the default local DB.
	adminDSN := "postgres://afi:afi@localhost:5433/afi?sslmode=disable"
	admin, adminErr := pgxpool.New(ctx, adminDSN)
	if adminErr != nil {
		return err
	}
	defer admin.Close()
	_, _ = admin.Exec(ctx, `CREATE DATABASE afi_test`)
	retry, retryErr := pgxpool.New(ctx, dsn)
	if retryErr != nil {
		return err
	}
	defer retry.Close()
	return retry.Ping(ctx)
}

func resetDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if err := controlplane.ResetDatabase(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
}

func seedUserOrg(t *testing.T, pool *pgxpool.Pool, userID, orgID string) {
	t.Helper()
	users := postgres.NewUsers(pool)
	orgs := postgres.NewOrganizations(pool)
	now := time.Now().UTC()
	if err := users.Create(t.Context(), identity.User{
		ID: userID, Email: userID + "@afi.test", Name: "User", Role: "admin",
		PasswordHash: "x", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := orgs.CreateWithOwner(t.Context(), tenancy.Organization{
		ID: orgID, Name: "Org", CreatedAt: now,
	}, userID); err != nil {
		t.Fatal(err)
	}
}

func TestOrganizationsCreateWithOwnerAndRoleChange(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	users := postgres.NewUsers(pool)
	orgs := postgres.NewOrganizations(pool)
	ctx := t.Context()
	now := time.Now().UTC()

	if err := users.Create(ctx, identity.User{
		ID: "u_owner", Email: "owner@afi.test", Name: "Owner", Role: "admin",
		PasswordHash: "x", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, identity.User{
		ID: "u_member", Email: "member@afi.test", Name: "Member", Role: "admin",
		PasswordHash: "x", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := orgs.CreateWithOwner(ctx, tenancy.Organization{
		ID: "org_1", Name: "Acme", CreatedAt: now,
	}, "u_owner"); err != nil {
		t.Fatal(err)
	}
	if err := orgs.AddMember(ctx, "org_1", "u_member", tenancy.OrgRoleMember); err != nil {
		t.Fatal(err)
	}

	got, err := orgs.Get(ctx, "org_1")
	if err != nil || got.Name != "Acme" {
		t.Fatalf("org=%+v err=%v", got, err)
	}
	role, err := orgs.GetMemberRole(ctx, "u_owner", "org_1")
	if err != nil || role != tenancy.OrgRoleOwner {
		t.Fatalf("role=%q err=%v", role, err)
	}
	n, err := orgs.CountOwners(ctx, "org_1")
	if err != nil || n != 1 {
		t.Fatalf("owners=%d err=%v", n, err)
	}

	if err := orgs.ApplyRoleChange(ctx, "org_1", "u_owner", "u_member", tenancy.OrgRoleOwner, true); err != nil {
		t.Fatal(err)
	}
	ownerRole, _ := orgs.GetMemberRole(ctx, "u_owner", "org_1")
	memberRole, _ := orgs.GetMemberRole(ctx, "u_member", "org_1")
	if ownerRole != tenancy.OrgRoleAdmin || memberRole != tenancy.OrgRoleOwner {
		t.Fatalf("after transfer owner=%q member=%q", ownerRole, memberRole)
	}
	if err := orgs.SetMailProvider(ctx, "org_1", "resend"); err != nil {
		t.Fatal(err)
	}
	got, _ = orgs.Get(ctx, "org_1")
	if got.MailProvider != "resend" {
		t.Fatalf("mail_provider=%q", got.MailProvider)
	}
}

func TestAPIKeysNullOwnerAndProjectScan(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	seedUserOrg(t, pool, "u1", "org_1")
	keys := postgres.NewAPIKeys(pool)
	ctx := t.Context()

	if err := keys.Insert(ctx, access.APIKey{
		ID: "key_sa", OrganizationID: "org_1", Name: "sa", Kind: "service_account",
		KeyPrefix: "sk-sa", CreatedAt: time.Now().UTC(),
	}, "hash_sa"); err != nil {
		t.Fatal(err)
	}
	got, err := keys.Get(ctx, "key_sa")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectID != "" || got.OwnerUserID != "" || got.Kind != "service_account" {
		t.Fatalf("key=%+v", got)
	}

	if err := keys.Insert(ctx, access.APIKey{
		ID: "key_personal", OrganizationID: "org_1", Name: "mine", Kind: "personal",
		OwnerUserID: "u1", KeyPrefix: "sk-me", CreatedAt: time.Now().UTC(),
	}, "hash_personal"); err != nil {
		t.Fatal(err)
	}
	personal, err := keys.Get(ctx, "key_personal")
	if err != nil {
		t.Fatal(err)
	}
	if personal.OwnerUserID != "u1" || personal.ProjectID != "" {
		t.Fatalf("personal=%+v", personal)
	}
}

func TestSnapshotPutLatestAndSourceLoad(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	seedUserOrg(t, pool, "u1", "org_1")
	ctx := t.Context()

	if _, err := pool.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities)
		VALUES ('prov_1', 'org_1', 'OpenAI', 'openai', 'https://api.openai.com/v1', 'OPENAI_API_KEY', '{"chat":true,"stream":true}')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, fallbacks, retry)
		VALUES (
			'route_1', 'org_1', 'gpt-4o-mini', 'prov_1', 'gpt-4o-mini',
			'[{"provider_id":"prov_1","target_model":"gpt-4o"}]',
			'{"max_attempts":3,"backoff":{"strategy":"exponential","base_delay":"100ms","max_delay":"1s","multiplier":2}}'
		)
	`); err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewAPIKeys(pool).Insert(ctx, access.APIKey{
		ID: "key_1", OrganizationID: "org_1", Name: "gw", Kind: "service_account",
		KeyPrefix: "sk-test", CreatedAt: time.Now().UTC(),
	}, "hash_1"); err != nil {
		t.Fatal(err)
	}

	loader := postgres.NewSnapshotSourceLoader(pool)
	src, err := loader.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.APIKeys) != 1 || len(src.Providers) != 1 || len(src.Routes) != 1 {
		t.Fatalf("src keys=%d providers=%d routes=%d", len(src.APIKeys), len(src.Providers), len(src.Routes))
	}
	if len(src.Routes[0].Fallbacks) != 1 || src.Routes[0].Fallbacks[0].TargetModel != "gpt-4o" {
		t.Fatalf("fallbacks=%+v", src.Routes[0].Fallbacks)
	}
	if src.Routes[0].Retry == nil || src.Routes[0].Retry.MaxAttempts != 3 || src.Routes[0].Retry.Backoff.Strategy != "exponential" {
		t.Fatalf("retry=%+v", src.Routes[0].Retry)
	}

	compiled := snapshot.Compile(src)
	store := postgres.NewSnapshotStore(pool)
	version, err := store.Put(ctx, compiled)
	if err != nil || version < 1 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	latest, err := store.Latest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != version {
		t.Fatalf("latest version=%d want %d", latest.Version, version)
	}
	if _, ok := latest.Providers["prov_1"]; !ok {
		t.Fatalf("providers=%v", latest.Providers)
	}
	if _, ok := latest.APIKeys["hash_1"]; !ok {
		t.Fatalf("api keys=%v", latest.APIKeys)
	}
}

func TestUsageOutboxClaimAndSink(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	ctx := t.Context()
	outbox := &postgres.UsageOutbox{Pool: pool}
	sink := &postgres.UsageSink{Pool: pool}
	prices := &postgres.PriceLookup{Pool: pool}

	if err := outbox.Enqueue(ctx, []byte(`{"model":"gpt-4o-mini"}`)); err != nil {
		t.Fatal(err)
	}
	rows, err := outbox.ClaimBatch(ctx, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if err := outbox.MarkProcessed(ctx, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	again, err := outbox.ClaimBatch(ctx, 10)
	if err != nil || len(again) != 0 {
		t.Fatalf("after mark: rows=%d err=%v", len(again), err)
	}

	cost := 0.01
	if err := sink.InsertUsage(ctx, usage.Event{
		OrganizationID: "org_1", ProjectID: "proj_1", APIKeyID: "key_1",
		Model: "gpt-4o-mini", Status: "ok", LatencyMs: 12,
		PromptTokens: 10, CompletionTokens: 5, Modality: "chat",
		Metrics: map[string]any{"requests": 1},
	}, &cost); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM usage_events`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("usage count=%d err=%v", n, err)
	}

	in, out, ok, err := prices.LookupModelPrice(ctx, "openai", "gpt-4o-mini")
	if err != nil || !ok || in <= 0 || out <= 0 {
		t.Fatalf("price in=%v out=%v ok=%v err=%v", in, out, ok, err)
	}
	_, _, ok, err = prices.LookupModelPrice(ctx, "openai", "missing-model")
	if err != nil || ok {
		t.Fatalf("missing price ok=%v err=%v", ok, err)
	}
}
