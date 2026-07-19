package gatewayconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type memMembers struct {
	projectOrg string
	memberOK   bool
	keyOrg     string
}

func (m memMembers) ProjectBelongsToOrg(context.Context, string, string) error {
	if m.projectOrg == "" {
		return fmtInvalid("project not found")
	}
	return nil
}

func (m memMembers) UserIsOrgMember(context.Context, string, string) error {
	if !m.memberOK {
		return fmtInvalid("user is not an organization member")
	}
	return nil
}

func (m memMembers) APIKeyBelongsToOrg(context.Context, string, string) error {
	if m.keyOrg == "" {
		return fmtInvalid("api key not found")
	}
	return nil
}

func fmtInvalid(msg string) error {
	return errors.Join(kernel.ErrInvalidRequest, errors.New(msg))
}

type memQuotas struct {
	items []Quota
}

func (m *memQuotas) ListByOrg(context.Context, string) ([]Quota, error) { return m.items, nil }
func (m *memQuotas) Insert(_ context.Context, q Quota) error {
	m.items = append(m.items, q)
	return nil
}
func (m *memQuotas) UpdateLimit(context.Context, string, int64) (*Quota, error) {
	return nil, kernel.ErrNotFound
}
func (m *memQuotas) Delete(context.Context, string) error { return nil }
func (m *memQuotas) OrgID(context.Context, string) (string, error) {
	return "", kernel.ErrNotFound
}

func TestCreateQuotaPersists(t *testing.T) {
	t.Parallel()
	repo := &memQuotas{}
	q, err := CreateQuota(context.Background(), repo, memMembers{memberOK: true}, "quota_1",
		"org_1", snapshot.ScopeUser, "user_1", snapshot.MetricRequests, 5, snapshot.WindowMinute)
	if err != nil {
		t.Fatal(err)
	}
	if q.ID != "quota_1" || len(repo.items) != 1 {
		t.Fatalf("q=%+v items=%d", q, len(repo.items))
	}
}

func TestCreateQuotaRejectsNonMember(t *testing.T) {
	t.Parallel()
	_, err := CreateQuota(context.Background(), &memQuotas{}, memMembers{memberOK: false}, "quota_1",
		"org_1", snapshot.ScopeUser, "user_1", snapshot.MetricRequests, 5, snapshot.WindowTotal)
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
