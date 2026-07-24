package grpcprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	sdkhook "github.com/curefatih/afi/sdk/hook"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
	"google.golang.org/grpc"
)

// Plugin is one loaded gRPC extension after a successful handshake.
type Plugin struct {
	Manifest Manifest
	Name     string
	Version  string
	Caps     []extensionv1.Capability

	Provider sdkprovider.ChatProvider
	Hooks    []any // BeforeCallHook / AfterCallHook / ChatHook / AfterChatHook

	conn *grpc.ClientConn
	proc *pluginProcess
}

// Runtime manages zero or more gRPC extension plugins.
type Runtime struct {
	log     *slog.Logger
	plugins []*Plugin
	closer  closerList
}

// Start loads each manifest (spawn or dial), handshakes, and builds adapters.
// Failures abort startup (fail-closed).
func Start(ctx context.Context, manifests []Manifest, log *slog.Logger) (*Runtime, error) {
	if log == nil {
		log = slog.Default()
	}
	rt := &Runtime{log: log}
	for _, m := range manifests {
		if err := m.validate(); err != nil {
			_ = rt.Close()
			return nil, err
		}
		p, err := rt.loadOne(ctx, m)
		if err != nil {
			_ = rt.Close()
			return nil, err
		}
		rt.plugins = append(rt.plugins, p)
	}
	return rt, nil
}

func (rt *Runtime) loadOne(ctx context.Context, m Manifest) (*Plugin, error) {
	var proc *pluginProcess
	target := strings.TrimSpace(m.Address)
	if target == "" {
		var err error
		proc, err = startProcess(ctx, m)
		if err != nil {
			return nil, fmt.Errorf("grpc extension %q: %w", m.ID, err)
		}
		rt.closer.add(proc.Close)
		target = "unix://" + proc.sock
	}

	conn, err := dialTarget(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("grpc extension %q dial: %w", m.ID, err)
	}
	rt.closer.add(conn.Close)

	ext := extensionv1.NewExtensionClient(conn)
	hs, err := ext.Handshake(ctx, &extensionv1.HandshakeRequest{HostVersion: hostProtocolVersion})
	if err != nil {
		return nil, fmt.Errorf("grpc extension %q handshake: %w", m.ID, err)
	}
	if hs == nil {
		return nil, fmt.Errorf("grpc extension %q: empty handshake", m.ID)
	}

	p := &Plugin{
		Manifest: m,
		Name:     firstNonEmpty(hs.GetName(), m.ID),
		Version:  hs.GetVersion(),
		Caps:     append([]extensionv1.Capability(nil), hs.GetCapabilities()...),
		conn:     conn,
		proc:     proc,
	}

	capSet := map[extensionv1.Capability]bool{}
	for _, c := range hs.GetCapabilities() {
		capSet[c] = true
	}

	hookClient := extensionv1.NewHookClient(conn)
	hookTimeout := m.hookTimeout()
	baseName := "grpc:" + m.ID

	if capSet[extensionv1.Capability_CAPABILITY_PROVIDER_CHAT] {
		typ, err := resolveProviderType(m, hs)
		if err != nil {
			return nil, err
		}
		p.Provider = newProviderAdapter(extensionv1.NewProviderClient(conn), typ, m.chatTimeout())
		rt.log.Info("grpc extension provider", "id", m.ID, "type", typ, "name", p.Name, "version", p.Version)
	}
	if capSet[extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CALL] {
		p.Hooks = append(p.Hooks, &BeforeCallAdapter{client: hookClient, name: baseName, timeout: hookTimeout})
	}
	if capSet[extensionv1.Capability_CAPABILITY_HOOK_AFTER_CALL] {
		p.Hooks = append(p.Hooks, &AfterCallAdapter{client: hookClient, name: baseName, timeout: hookTimeout})
	}
	if capSet[extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CHAT] {
		p.Hooks = append(p.Hooks, &BeforeChatAdapter{client: hookClient, name: baseName + ":before_chat", timeout: hookTimeout})
	}
	if capSet[extensionv1.Capability_CAPABILITY_HOOK_AFTER_CHAT] {
		p.Hooks = append(p.Hooks, &AfterChatAdapter{client: hookClient, name: baseName + ":after_chat", timeout: hookTimeout})
	}

	if p.Provider == nil && len(p.Hooks) == 0 {
		rt.log.Warn("grpc extension advertised no wired capabilities", "id", m.ID, "capabilities", hs.GetCapabilities())
	}
	return p, nil
}

// Plugins returns loaded plugins in config order.
func (rt *Runtime) Plugins() []*Plugin {
	if rt == nil {
		return nil
	}
	return rt.plugins
}

// Providers returns ChatProviders from plugins that advertised PROVIDER_CHAT.
func (rt *Runtime) Providers() []sdkprovider.ChatProvider {
	if rt == nil {
		return nil
	}
	var out []sdkprovider.ChatProvider
	for _, p := range rt.plugins {
		if p.Provider != nil {
			out = append(out, p.Provider)
		}
	}
	return out
}

// ApplyHooks registers plugin hooks onto a dataplane-compatible hook registrar.
func (rt *Runtime) ApplyHooks(register func(h any)) {
	if rt == nil || register == nil {
		return
	}
	for _, p := range rt.plugins {
		for _, h := range p.Hooks {
			register(h)
		}
	}
}

// Close shuts down connections and child processes.
func (rt *Runtime) Close() error {
	if rt == nil {
		return nil
	}
	return rt.closer.closeAll()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Ensure hook adapters satisfy SDK interfaces.
var (
	_ sdkhook.BeforeCallHook = (*BeforeCallAdapter)(nil)
	_ sdkhook.AfterCallHook  = (*AfterCallAdapter)(nil)
	_ sdkhook.ChatHook       = (*BeforeChatAdapter)(nil)
	_ sdkhook.AfterChatHook  = (*AfterChatAdapter)(nil)
)
