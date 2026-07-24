// Package grpcprovider hosts process-isolated gRPC extensions for the gateway.
//
// Plugins speak proto/afi/extension/v1 (Handshake + optional Provider/Hook).
// Discovery is gateway config only in v1 (no control-plane CRUD).
package grpcprovider

import (
	"fmt"
	"strings"
	"time"
)

// EnvPluginSock is set by the host when spawning a command-based plugin.
// The plugin must listen on this unix socket path (or the path after unix://).
const EnvPluginSock = "AFI_PLUGIN_SOCK"

const (
	hostProtocolVersion = "1"
	defaultHookTimeout  = 5 * time.Second
	defaultChatTimeout  = 120 * time.Second
	defaultDialTimeout  = 10 * time.Second
)

// Manifest describes one gRPC extension process.
type Manifest struct {
	// ID is a stable label for logs and hook names (required).
	ID string `yaml:"id" json:"id"`
	// Address is a dial target: unix:///path/to.sock or host:port.
	// Mutually exclusive with Command.
	Address string `yaml:"address" json:"address"`
	// Command is an argv used to spawn the plugin; the host sets AFI_PLUGIN_SOCK.
	Command []string `yaml:"command" json:"command"`
	// ProviderType overrides HandshakeResponse.provider_type when non-empty.
	ProviderType string `yaml:"provider_type" json:"provider_type"`
	// HookTimeout overrides the default BeforeCall/AfterCall/chat hook deadline.
	HookTimeout time.Duration `yaml:"-" json:"-"`
	// ChatTimeout overrides the default Chat RPC deadline.
	ChatTimeout time.Duration `yaml:"-" json:"-"`
}

func (m Manifest) validate() error {
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return fmt.Errorf("grpc extension: id is required")
	}
	addr := strings.TrimSpace(m.Address)
	hasCmd := len(m.Command) > 0 && strings.TrimSpace(m.Command[0]) != ""
	if addr != "" && hasCmd {
		return fmt.Errorf("grpc extension %q: set address or command, not both", id)
	}
	if addr == "" && !hasCmd {
		return fmt.Errorf("grpc extension %q: address or command is required", id)
	}
	return nil
}

func (m Manifest) hookTimeout() time.Duration {
	if m.HookTimeout > 0 {
		return m.HookTimeout
	}
	return defaultHookTimeout
}

func (m Manifest) chatTimeout() time.Duration {
	if m.ChatTimeout > 0 {
		return m.ChatTimeout
	}
	return defaultChatTimeout
}
