package gatewayconfig

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestNewMCPBackend(t *testing.T) {
	b, err := NewMCPBackend("mcp_1", "o1", "Docs", "Docs MCP", "https://mcp.example.com/mcp", "MCP_KEY", json.RawMessage(`["tools/list"]`), true, timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if b.Alias != "docs" {
		t.Fatalf("alias=%q", b.Alias)
	}
	if b.BaseURL != "https://mcp.example.com/mcp" {
		t.Fatalf("base_url=%q", b.BaseURL)
	}
}

func TestParseMCPAliasRejectsInvalid(t *testing.T) {
	if _, err := ParseMCPAlias("Bad Alias"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if _, err := ParseMCPAlias(""); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestParseMCPBaseURLRejectsRelative(t *testing.T) {
	if _, err := ParseMCPBaseURL("/relative"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
