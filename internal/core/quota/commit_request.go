package quota

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/usage"
)

type CommitRequest struct {
	Principal *auth.Principal

	Usage *usage.Report
}
