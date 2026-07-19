package gatewayconfig

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

type okValidator struct{}

func (okValidator) Validate(string) error { return nil }

type badValidator struct{}

func (badValidator) Validate(string) error { return errors.New("bad cel") }

func TestNewRequestPolicy(t *testing.T) {
	t.Parallel()
	p, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", true, 10, timeNowUTC(), okValidator{})
	if err != nil || p.Name != "allow" {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestNewRequestPolicyRejectsBadCEL(t *testing.T) {
	t.Parallel()
	_, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", true, 1, timeNowUTC(), badValidator{})
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
