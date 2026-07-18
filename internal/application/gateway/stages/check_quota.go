package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/quota"
)

type CheckQuota struct {
	service quota.Service
}

func NewCheckQuota(
	service quota.Service,
) *CheckQuota {
	return &CheckQuota{
		service: service,
	}
}

func (s *CheckQuota) Name() string {
	return "check_quota"
}

func (s *CheckQuota) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	req := &quota.Request{
		Principal: state.Principal(),
		Model:     state.Model(),
		Usage:     state.Usage(),
		Cost:      state.Cost(),
	}

	decision, err := s.service.Check(ctx, req)
	if err != nil {
		return err
	}

	state.SetDecision(decision)

	return nil
}
