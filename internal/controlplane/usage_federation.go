package controlplane

import (
	"context"
	"sort"
	"strings"

	"github.com/curefatih/afi/internal/adapters/federationclient"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/usage"
)

func (s *Server) observePulledUsage(ctx context.Context, peerID string, reports []usage.Record) {
	for _, rep := range reports {
		region := ""
		if rep.Tags != nil {
			region = rep.Tags["region"]
		}
		if s.Metrics != nil {
			s.Metrics.RecordSpokeUsage(ctx, region, rep.Status, usage.NormalizeModality(rep.Modality), rep.PromptTokens, rep.CompletionTokens)
		}
		if s.log != nil {
			s.log.Info("pulled spoke usage report",
				"peer_id", peerID,
				"region", region,
				"org", rep.OrganizationID,
				"model", rep.Model,
				"status", rep.Status,
				"prompt_tokens", rep.PromptTokens,
				"completion_tokens", rep.CompletionTokens,
			)
		}
	}
}

func (s *Server) federationHomeMode() bool {
	return s != nil && s.cfg != nil && s.cfg.Federation.Mode == "home"
}

// fetchFederationUsageForOrg pulls regional usage reports for an org (observe-only; no hub persist).
func (s *Server) fetchFederationUsageForOrg(ctx context.Context, orgID string, f UsageFilter) []UsageEvent {
	if !s.federationHomeMode() || s.app == nil || strings.TrimSpace(orgID) == "" {
		return nil
	}
	peers, err := s.app.ListFederationPeers(ctx)
	if err != nil || len(peers) == 0 {
		return nil
	}
	limit := 500
	var out []UsageEvent
	for _, peer := range peers {
		if peer.Status == federation.PeerStatusDisabled {
			continue
		}
		if strings.TrimSpace(peer.BaseURL) == "" {
			continue
		}
		token, err := s.app.OpenFederationPeerJoinToken(ctx, peer.ID)
		if err != nil || token == "" {
			if s.log != nil {
				s.log.Warn("skip federation usage pull", "peer_id", peer.ID, "err", err)
			}
			continue
		}
		client := federationclient.New(peer.BaseURL, token)
		resp, err := client.UsageReports(ctx, orgID, f.From, f.To, limit)
		if err != nil {
			if s.log != nil {
				s.log.Warn("federation usage pull failed", "peer_id", peer.ID, "err", err)
			}
			continue
		}
		s.observePulledUsage(ctx, peer.ID, resp.Reports)
		for _, rep := range resp.Reports {
			if !usageMatchesFilter(rep, orgID, f) {
				continue
			}
			if rep.Tags == nil {
				rep.Tags = map[string]string{}
			}
			if rep.Tags["region"] == "" {
				if reg, err := s.app.GetRegion(ctx, peer.RegionID); err == nil && reg != nil {
					rep.Tags["region"] = reg.Slug
				}
			}
			rep.Tags["federation_peer_id"] = peer.ID
			out = append(out, rep)
		}
	}
	return out
}

func usageMatchesFilter(e usage.Record, orgID string, f UsageFilter) bool {
	if e.OrganizationID != orgID {
		return false
	}
	if f.ProjectID != "" && e.ProjectID != f.ProjectID {
		return false
	}
	if f.APIKeyID != "" && e.APIKeyID != f.APIKeyID {
		return false
	}
	if f.CredentialID != "" && e.CredentialID != f.CredentialID {
		return false
	}
	if f.Model != "" && e.Model != f.Model {
		return false
	}
	if f.Modality != "" && usage.NormalizeModality(e.Modality) != usage.NormalizeModality(f.Modality) {
		return false
	}
	if f.ExcludeBYOK && e.UsedBYOK {
		return false
	}
	if f.BYOKOnly && !e.UsedBYOK {
		return false
	}
	if f.From != nil && e.CreatedAt.Before(f.From.UTC()) {
		return false
	}
	if f.To != nil && !e.CreatedAt.Before(f.To.UTC()) {
		return false
	}
	return true
}

func mergeUsageEvents(local, remote []UsageEvent, limit int) []UsageEvent {
	merged := make([]UsageEvent, 0, len(local)+len(remote))
	merged = append(merged, local...)
	merged = append(merged, remote...)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})
	if limit <= 0 {
		limit = 50
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

func summarizeUsageRecords(records []UsageEvent, groupBy string) []UsageSummaryBucket {
	type acc struct {
		bucket           string
		label            string
		requests         int64
		cost             float64
		promptTokens     int64
		completionTokens int64
		metrics          map[string]float64
		keyKind          string
		ownerEmail       string
		ownerName        string
	}
	by := map[string]*acc{}
	for _, e := range records {
		b, label := usageSummaryBucketKey(e, groupBy)
		a := by[b]
		if a == nil {
			a = &acc{bucket: b, label: label, metrics: map[string]float64{}, keyKind: e.KeyKind, ownerEmail: e.OwnerEmail, ownerName: e.OwnerName}
			by[b] = a
		}
		a.requests++
		a.promptTokens += e.PromptTokens
		a.completionTokens += e.CompletionTokens
		if e.CostUSD != nil {
			a.cost += *e.CostUSD
		}
		for k, v := range e.Metrics {
			switch n := v.(type) {
			case float64:
				a.metrics[k] += n
			case int:
				a.metrics[k] += float64(n)
			case int64:
				a.metrics[k] += float64(n)
			}
		}
	}
	out := make([]UsageSummaryBucket, 0, len(by))
	for _, a := range by {
		out = append(out, UsageSummaryBucket{
			Bucket:           a.bucket,
			Label:            a.label,
			Requests:         a.requests,
			CostUSD:          a.cost,
			PromptTokens:     a.promptTokens,
			CompletionTokens: a.completionTokens,
			MetricsTotals:    a.metrics,
			KeyKind:          a.keyKind,
			OwnerEmail:       a.ownerEmail,
			OwnerName:        a.ownerName,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if groupBy == "day" {
			return out[i].Bucket < out[j].Bucket
		}
		if out[i].Requests != out[j].Requests {
			return out[i].Requests > out[j].Requests
		}
		return out[i].Bucket < out[j].Bucket
	})
	return out
}

func usageSummaryBucketKey(e UsageEvent, groupBy string) (bucket, label string) {
	switch groupBy {
	case "model":
		return e.Model, e.Model
	case "key":
		id := e.APIKeyID
		if id == "" {
			id = "unknown"
		}
		label = e.KeyName
		if label == "" {
			label = id
		}
		return id, label
	case "modality":
		m := usage.NormalizeModality(e.Modality)
		return m, m
	case "byok":
		if e.UsedBYOK {
			return "byok", "BYOK"
		}
		return "platform", "Platform"
	default: // day
		day := e.CreatedAt.UTC().Format("2006-01-02")
		return day, day
	}
}

func mergeUsageSummary(local, remote []UsageSummaryBucket) []UsageSummaryBucket {
	by := map[string]UsageSummaryBucket{}
	order := []string{}
	add := func(b UsageSummaryBucket) {
		cur, ok := by[b.Bucket]
		if !ok {
			by[b.Bucket] = b
			order = append(order, b.Bucket)
			return
		}
		cur.Requests += b.Requests
		cur.CostUSD += b.CostUSD
		cur.PromptTokens += b.PromptTokens
		cur.CompletionTokens += b.CompletionTokens
		if cur.MetricsTotals == nil {
			cur.MetricsTotals = map[string]float64{}
		}
		for k, v := range b.MetricsTotals {
			cur.MetricsTotals[k] += v
		}
		if cur.Label == "" {
			cur.Label = b.Label
		}
		by[b.Bucket] = cur
	}
	for _, b := range local {
		add(b)
	}
	for _, b := range remote {
		add(b)
	}
	out := make([]UsageSummaryBucket, 0, len(order))
	for _, k := range order {
		out = append(out, by[k])
	}
	sort.Slice(out, func(i, j int) bool {
		// Prefer chronological for day buckets (YYYY-MM-DD).
		if len(out[i].Bucket) == 10 && out[i].Bucket[4] == '-' && out[j].Bucket[4] == '-' {
			return out[i].Bucket < out[j].Bucket
		}
		if out[i].Requests != out[j].Requests {
			return out[i].Requests > out[j].Requests
		}
		return out[i].Bucket < out[j].Bucket
	})
	return out
}
