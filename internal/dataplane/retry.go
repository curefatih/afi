package dataplane

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
)

// maxTriesFor returns how many times to call a single route target (includes the first try).
func maxTriesFor(cfg *snapshot.RetryConfig) int {
	if cfg == nil || cfg.MaxAttempts < 1 {
		return 1
	}
	return cfg.MaxAttempts
}

// sleepBeforeRetry waits for the configured backoff before retry index n (0 = first retry).
// Returns ctx.Err() when the request is cancelled during the wait.
func sleepBeforeRetry(ctx context.Context, cfg *snapshot.RetryConfig, n int) error {
	if cfg == nil || n < 0 {
		return nil
	}
	d := cfg.Delay(n)
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func discardResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func logRetry(log *slog.Logger, providerID string, try, maxTries, status int, err error) {
	if log == nil {
		return
	}
	if err != nil {
		log.Warn("upstream retry", "provider", providerID, "try", try+1, "max_tries", maxTries, "err", err)
		return
	}
	log.Warn("upstream retry", "provider", providerID, "try", try+1, "max_tries", maxTries, "status", status)
}
