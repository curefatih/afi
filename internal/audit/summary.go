package audit

import (
	"fmt"
	"strings"
)

// Summary builds a short human-readable description for an audit entry.
func Summary(name, resourceID string) string {
	res := strings.TrimSpace(resourceID)
	switch name {
	case "org.created":
		return fmt.Sprintf("Created organization %s", res)
	case "member.added":
		return fmt.Sprintf("Added member %s", res)
	case "member.role_updated":
		return fmt.Sprintf("Updated member role for %s", res)
	case "invite.created":
		return fmt.Sprintf("Created invite %s", res)
	case "invite.revoked":
		return fmt.Sprintf("Revoked invite %s", res)
	case "invite.resent":
		return fmt.Sprintf("Resent invite %s", res)
	case "invite.accepted":
		return fmt.Sprintf("Accepted invite (user %s)", res)
	case "team.created":
		return fmt.Sprintf("Created team %s", res)
	case "team.member_added":
		return fmt.Sprintf("Added team member %s", res)
	case "team.member_role_updated":
		return fmt.Sprintf("Updated team member role for %s", res)
	case "team.member_removed":
		return fmt.Sprintf("Removed team member %s", res)
	case "project.created":
		return fmt.Sprintf("Created project %s", res)
	case "api_key.created":
		return fmt.Sprintf("Created API key %s", res)
	case "api_key.deleted":
		return fmt.Sprintf("Deleted API key %s", res)
	case "provider.created":
		return fmt.Sprintf("Created provider %s", res)
	case "provider.updated":
		return fmt.Sprintf("Updated provider %s", res)
	case "provider.deleted":
		return fmt.Sprintf("Deleted provider %s", res)
	case "route.created":
		return fmt.Sprintf("Created route %s", res)
	case "route.updated":
		return fmt.Sprintf("Updated route %s", res)
	case "route.deleted":
		return fmt.Sprintf("Deleted route %s", res)
	case "org.default_retry.updated":
		return "Updated organization default retry"
	case "quota.created":
		return fmt.Sprintf("Created quota %s", res)
	case "quota.updated":
		return fmt.Sprintf("Updated quota %s", res)
	case "quota.deleted":
		return fmt.Sprintf("Deleted quota %s", res)
	case "policy.created":
		return fmt.Sprintf("Created policy %s", res)
	case "policy.updated":
		return fmt.Sprintf("Updated policy %s", res)
	case "policy.deleted":
		return fmt.Sprintf("Deleted policy %s", res)
	case "wasm_hook.created":
		return fmt.Sprintf("Created WASM hook %s", res)
	case "wasm_hook.updated":
		return fmt.Sprintf("Updated WASM hook %s", res)
	case "wasm_hook.deleted":
		return fmt.Sprintf("Deleted WASM hook %s", res)
	case "mcp_backend.created":
		return fmt.Sprintf("Created MCP backend %s", res)
	case "mcp_backend.updated":
		return fmt.Sprintf("Updated MCP backend %s", res)
	case "mcp_backend.deleted":
		return fmt.Sprintf("Deleted MCP backend %s", res)
	case "a2a_agent.created":
		return fmt.Sprintf("Created A2A agent %s", res)
	case "a2a_agent.updated":
		return fmt.Sprintf("Updated A2A agent %s", res)
	case "a2a_agent.deleted":
		return fmt.Sprintf("Deleted A2A agent %s", res)
	case "credential.created":
		return fmt.Sprintf("Created credential %s", res)
	case "credential.updated":
		return fmt.Sprintf("Updated credential %s", res)
	case "credential.rotated":
		return fmt.Sprintf("Rotated credential %s", res)
	case "credential.deleted":
		return fmt.Sprintf("Deleted credential %s", res)
	case "credential.assigned":
		return fmt.Sprintf("Assigned credential %s", res)
	case "credential.unassigned":
		return fmt.Sprintf("Unassigned credential %s", res)
	case "snapshot.published":
		return "Published gateway snapshot"
	default:
		if res != "" {
			return fmt.Sprintf("%s (%s)", name, res)
		}
		return name
	}
}
