package gatewayconfig

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestNewA2AAgent(t *testing.T) {
	a, err := NewA2AAgent("a2a_1", "o1", "Helper", "Helper Agent", "https://agent.example/rpc", "", "A2A_KEY", "bearer", nil, true, timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if a.Alias != "helper" {
		t.Fatalf("alias=%q", a.Alias)
	}
	if a.UpstreamURL != "https://agent.example/rpc" {
		t.Fatalf("upstream=%q", a.UpstreamURL)
	}
}

func TestParseA2AAliasRejectsInvalid(t *testing.T) {
	if _, err := ParseA2AAlias("Bad Alias"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
