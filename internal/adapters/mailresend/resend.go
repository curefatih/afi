package mailresend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/mail"
)

const defaultAPIURL = "https://api.resend.com/emails"

// Config holds Resend API settings.
type Config struct {
	APIKey  string
	From    string
	BaseURL string // optional; defaults to Resend production API (tests may override)
}

// Sender delivers mail via the Resend HTTP API.
type Sender struct {
	Cfg    Config
	Client *http.Client
}

func (s Sender) Send(ctx context.Context, msg mail.Message) error {
	if strings.TrimSpace(s.Cfg.APIKey) == "" {
		return fmt.Errorf("resend api key required")
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	body := map[string]any{
		"from":    s.Cfg.From,
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"text":    msg.TextBody,
		"html":    msg.HTMLBody,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	apiURL := strings.TrimSpace(s.Cfg.BaseURL)
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return fmt.Errorf("resend: status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
}
