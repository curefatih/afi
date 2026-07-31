package identity

// Platform user roles (users.role). Distinct from org/team membership roles.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// IsPlatformAdmin reports whether the user may manage platform topology (regions/deployments).
func IsPlatformAdmin(role string) bool {
	return role == RoleAdmin
}
