package dataplane

import (
	"net/http"

	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

// checkPolicies evaluates CEL allow-policies. ok=false means the response was written.
func (p *Pipeline) checkPolicies(w http.ResponseWriter, snap *snapshot.Snapshot, key snapshot.APIKey, model, path string, stream bool) bool {
	if p.Policies == nil || snap == nil || len(snap.Policies) == 0 {
		return true
	}
	allowed, deniedName, err := p.Policies.Evaluate(snap.Policies, key, policy.Request{
		Model:  model,
		Path:   path,
		Stream: stream,
	})
	if err != nil {
		if p.Log != nil {
			p.Log.Error("policy check", "err", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "policy check failed", "type": "server_error"},
		})
		return false
	}
	if !allowed {
		msg := "request denied by policy"
		if deniedName != "" {
			msg = "request denied by policy: " + deniedName
		}
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]string{
				"message": msg,
				"type":    "policy_violation",
				"code":    "policy_violation",
			},
		})
		return false
	}
	return true
}
