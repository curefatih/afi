package tenancy

// OrgRole is a typed organization membership role.
type OrgRole string

// ParseOrgRole validates an organization role string.
func ParseOrgRole(role string) (OrgRole, error) {
	if err := ValidateOrgRole(role); err != nil {
		return "", err
	}
	return OrgRole(role), nil
}

func (r OrgRole) String() string { return string(r) }

// IsAdmin reports whether the role grants admin privileges.
func (r OrgRole) IsAdmin() bool {
	return r == OrgRole(OrgRoleOwner) || r == OrgRole(OrgRoleAdmin)
}

// CanChangeRoles reports whether the role may change other members' roles.
func (r OrgRole) CanChangeRoles() bool {
	return r == OrgRole(OrgRoleOwner)
}
