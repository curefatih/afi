package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/quota"
)

type CommitUsage struct {
	service *quota.Service
}

func NewCommitUsage(
	service *quota.Service,
) *CommitUsage {
	return &CommitUsage{
		service: service,
	}
}

func (s *CommitUsage) Name() string {
	return "commit_usage"
}

func (s *CommitUsage) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	if state.Usage == nil {
		return nil
	}

	req := &quota.CommitRequest{
		Principal: state.Principal(),
		Usage:     state.Usage(),
	}

	return s.service.Commit(ctx, req)
}
