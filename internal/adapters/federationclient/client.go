package federationclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/usage"
)

const TokenHeader = "X-AFI-Federation-Token"

// Client pulls region exports from a home control plane.
type Client struct {
	BaseURL    string
	JoinToken  string
	HTTPClient *http.Client
}

func New(baseURL, joinToken string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		JoinToken:  strings.TrimSpace(joinToken),
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Join exchanges the join token for peer identity on the hub.
func (c *Client) Join(ctx context.Context) (*federation.ControlPlanePeer, error) {
	if c == nil || c.BaseURL == "" || c.JoinToken == "" {
		return nil, fmt.Errorf("federationclient: join not configured")
	}
	body, _ := json.Marshal(map[string]string{"join_token": c.JoinToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/internal/v1/federation/peers/join", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("federationclient join: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var peer federation.ControlPlanePeer
	if err := json.Unmarshal(b, &peer); err != nil {
		return nil, err
	}
	return &peer, nil
}

// Export pulls a region export document (optionally since a revision).
func (c *Client) Export(ctx context.Context, regionSlug string, since int64) (*federation.RegionExport, error) {
	if c == nil || c.BaseURL == "" || c.JoinToken == "" {
		return nil, fmt.Errorf("federationclient: export not configured")
	}
	regionSlug = strings.TrimSpace(regionSlug)
	u := c.BaseURL + "/internal/v1/federation/regions/" + url.PathEscape(regionSlug) + "/export"
	if since > 0 {
		u += "?since=" + strconv.FormatInt(since, 10)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(TokenHeader, c.JoinToken)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("federationclient export: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var exp federation.RegionExport
	if err := json.Unmarshal(b, &exp); err != nil {
		return nil, err
	}
	return &exp, nil
}

// UsageReportsResponse is returned by regional usage report pull.
type UsageReportsResponse struct {
	Reports []usage.Record `json:"reports"`
}

// UsageReports pulls usage events from a regional control plane (hub on-demand).
func (c *Client) UsageReports(ctx context.Context, since *time.Time, limit int) (*UsageReportsResponse, error) {
	if c == nil || c.BaseURL == "" || c.JoinToken == "" {
		return nil, fmt.Errorf("federationclient: usage reports not configured")
	}
	u := c.BaseURL + "/internal/v1/federation/usage-reports"
	q := url.Values{}
	if since != nil && !since.IsZero() {
		q.Set("since", since.UTC().Format(time.RFC3339))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(TokenHeader, c.JoinToken)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("federationclient usage-reports: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out UsageReportsResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out.Reports == nil {
		out.Reports = []usage.Record{}
	}
	return &out, nil
}
