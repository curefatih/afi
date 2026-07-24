package grpcprovider

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type pluginProcess struct {
	cmd    *exec.Cmd
	sock   string
	cancel context.CancelFunc
	done   <-chan error
}

func startProcess(ctx context.Context, m Manifest) (*pluginProcess, error) {
	if len(m.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	dir, err := os.MkdirTemp("", "afi-grpc-"+sanitizeID(m.ID)+"-")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	sock := filepath.Join(dir, "plugin.sock")

	pctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(pctx, m.Command[0], m.Command[1:]...)
	cmd.Env = append(os.Environ(), EnvPluginSock+"="+sock)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("start %q: %w", m.Command[0], err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		close(done)
	}()

	p := &pluginProcess{cmd: cmd, sock: sock, cancel: cancel, done: done}
	if err := waitForSocket(pctx, sock, defaultDialTimeout); err != nil {
		_ = p.Close()
		return nil, err
	}
	return p, nil
}

func (p *pluginProcess) Close() error {
	if p == nil {
		return nil
	}
	if p.cancel != nil {
		p.cancel()
	}
	var err error
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
		select {
		case err = <-p.done:
		case <-time.After(2 * time.Second):
			_ = p.cmd.Process.Kill()
			err = <-p.done
		}
	}
	if p.sock != "" {
		dir := filepath.Dir(p.sock)
		_ = os.RemoveAll(dir)
	}
	return err
}

func waitForSocket(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for plugin socket %s", path)
		}
		fi, err := os.Stat(path)
		if err == nil && fi.Mode()&os.ModeSocket != 0 {
			// Brief settle so Accept is ready.
			conn, dialErr := net.DialTimeout("unix", path, 200*time.Millisecond)
			if dialErr == nil {
				_ = conn.Close()
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func dialTarget(ctx context.Context, target string) (*grpc.ClientConn, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("empty dial target")
	}

	var dialTarget string
	switch {
	case strings.HasPrefix(target, "unix://"):
		dialTarget = "unix:" + strings.TrimPrefix(target, "unix://")
	case strings.HasPrefix(target, "unix:"):
		dialTarget = target
	case filepath.IsAbs(target) || strings.HasPrefix(target, "/"):
		dialTarget = "unix:" + target
	default:
		dialTarget = target
	}

	conn, err := grpc.NewClient(dialTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	conn.Connect()

	dctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return conn, nil
		}
		if state == connectivity.Shutdown {
			_ = conn.Close()
			return nil, fmt.Errorf("connection shutdown before ready")
		}
		if !conn.WaitForStateChange(dctx, state) {
			_ = conn.Close()
			return nil, fmt.Errorf("dial timeout waiting for ready: %w", dctx.Err())
		}
	}
}

func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	s := b.String()
	if s == "" {
		return "plugin"
	}
	return s
}

// closerList closes multiple resources.
type closerList struct {
	mu      sync.Mutex
	closers []func() error
}

func (c *closerList) add(fn func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closers = append(c.closers, fn)
}

func (c *closerList) closeAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var first error
	for i := len(c.closers) - 1; i >= 0; i-- {
		if err := c.closers[i](); err != nil && first == nil {
			first = err
		}
	}
	c.closers = nil
	return first
}
