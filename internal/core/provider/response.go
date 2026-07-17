package provider

import "github.com/curefatih/afi/internal/core/usage"

type Response struct {
	Body any

	Usage *usage.Report
}
