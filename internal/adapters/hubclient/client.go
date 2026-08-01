package hubclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the hub control plane for spoke operations.
type Client struct {
	BaseURL    string
	JoinToken  string
	HTTPClient *http.Client
}

func New(baseURL, joinToken string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		JoinToken:  strings.TrimSpace(joinToken),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Heartbeat reports snapshot version for a deployment.
func (c *Client) Heartbeat(ctx context.Context, deploymentID string, snapVersion int64, build string) error {
	if c == nil || c.BaseURL == "" || c.JoinToken == "" || deploymentID == "" {
		return fmt.Errorf("hubclient: heartbeat not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"snapshot_version": snapVersion,
		"build":            build,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/internal/v1/deployments/"+deploymentID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AFI-Deployment-Token", c.JoinToken)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("hubclient heartbeat: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// ShipUsage posts a usage outbox payload to the hub.
func (c *Client) ShipUsage(ctx context.Context, deploymentID string, payload []byte) error {
	if c == nil || c.BaseURL == "" || c.JoinToken == "" || deploymentID == "" {
		return fmt.Errorf("hubclient: usage ship not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/internal/v1/deployments/"+deploymentID+"/usage", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AFI-Deployment-Token", c.JoinToken)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("hubclient usage: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// RunHeartbeatLoop sends heartbeats until ctx is cancelled.
func RunHeartbeatLoop(ctx context.Context, c *Client, deploymentID string, interval time.Duration, versionFn func() int64, build string, onErr func(error)) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	send := func() {
		if err := c.Heartbeat(ctx, deploymentID, versionFn(), build); err != nil && onErr != nil {
			onErr(err)
		}
	}
	send()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
