package gatewayconfig

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestNewProviderRequiresNameType(t *testing.T) {
	t.Parallel()
	_, err := NewProvider("p1", "o1", "", "openai", "https://x", "KEY", snapshot.ProviderCapabilities{}, timeNowUTC())
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestNewRouteRequiresModel(t *testing.T) {
	t.Parallel()
	_, err := NewRoute("r1", "o1", "", "prov", "m", nil, nil, "", 0, timeNowUTC())
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
