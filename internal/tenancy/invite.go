package tenancy

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

const defaultInviteTTL = 7 * 24 * time.Hour

// HashInviteToken returns the hex-encoded SHA-256 of the raw token.
func HashInviteToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewInviteToken generates an opaque invite token and its hash.
func NewInviteToken() (raw, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b[:])
	return raw, HashInviteToken(raw), nil
}

// NormalizeEmail trims and lowercases an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// InviteOrgMember adds an existing user or creates/resends a pending invite.
// When Status is "invited", rawToken is the plaintext accept token (shown once).
func InviteOrgMember(
	ctx context.Context,
	orgs OrganizationRepository,
	invites InviteRepository,
	users UserLookup,
	inviteID, orgID, email, invitedByUserID string,
) (outcome *InviteOutcome, rawToken string, err error) {
	email = NormalizeEmail(email)
	if orgID == "" || email == "" || invitedByUserID == "" {
		return nil, "", kernel.ErrInvalidRequest
	}
	if !strings.Contains(email, "@") {
		return nil, "", kernel.ErrInvalidRequest
	}

	userID, name, userEmail, lookupErr := users.FindByEmail(ctx, email)
	if lookupErr == nil {
		if _, err := orgs.GetMemberRole(ctx, userID, orgID); err == nil {
			return nil, "", fmt.Errorf("%w: already a member", kernel.ErrInvalidRequest)
		} else if !errors.Is(err, kernel.ErrNotFound) {
			return nil, "", err
		}
		if err := orgs.AddMember(ctx, orgID, userID, OrgRoleMember); err != nil {
			return nil, "", err
		}
		return &InviteOutcome{
			Status: "added",
			Member: &OrgMember{UserID: userID, Email: userEmail, Name: name, Role: OrgRoleMember},
		}, "", nil
	}
	if !errors.Is(lookupErr, kernel.ErrNotFound) {
		return nil, "", lookupErr
	}

	raw, hash, err := NewInviteToken()
	if err != nil {
		return nil, "", err
	}
	now := timeNowUTC()
	expires := now.Add(defaultInviteTTL)

	if existing, err := invites.GetPendingByOrgEmail(ctx, orgID, email); err == nil {
		if err := invites.UpdateToken(ctx, existing.ID, hash, expires); err != nil {
			return nil, "", err
		}
		existing.ExpiresAt = expires
		existing.Status = InviteStatusPending
		return &InviteOutcome{Status: "invited", Invite: existing}, raw, nil
	} else if !errors.Is(err, kernel.ErrNotFound) {
		return nil, "", err
	}

	if inviteID == "" {
		return nil, "", kernel.ErrInvalidRequest
	}
	inv := OrgInvite{
		ID:              inviteID,
		OrganizationID:  orgID,
		Email:           email,
		Role:            OrgRoleMember,
		InvitedByUserID: invitedByUserID,
		Status:          InviteStatusPending,
		ExpiresAt:       expires,
		CreatedAt:       now,
	}
	if err := invites.Insert(ctx, inv, hash); err != nil {
		return nil, "", err
	}
	return &InviteOutcome{Status: "invited", Invite: &inv}, raw, nil
}

// RevokeInvite marks a pending invite revoked.
func RevokeInvite(ctx context.Context, invites InviteRepository, orgID, inviteID string) error {
	inv, err := invites.Get(ctx, inviteID)
	if err != nil {
		return err
	}
	if inv.OrganizationID != orgID {
		return kernel.ErrNotFound
	}
	if inv.Status != InviteStatusPending {
		return kernel.ErrInvalidRequest
	}
	return invites.MarkRevoked(ctx, inviteID)
}

// ResendInvite rotates the token for a pending invite.
func ResendInvite(ctx context.Context, invites InviteRepository, orgID, inviteID string) (inv *OrgInvite, rawToken string, err error) {
	existing, err := invites.Get(ctx, inviteID)
	if err != nil {
		return nil, "", err
	}
	if existing.OrganizationID != orgID || existing.Status != InviteStatusPending {
		return nil, "", kernel.ErrInvalidRequest
	}
	if timeNowUTC().After(existing.ExpiresAt) {
		return nil, "", kernel.ErrInvalidRequest
	}
	raw, hash, err := NewInviteToken()
	if err != nil {
		return nil, "", err
	}
	expires := timeNowUTC().Add(defaultInviteTTL)
	if err := invites.UpdateToken(ctx, inviteID, hash, expires); err != nil {
		return nil, "", err
	}
	existing.ExpiresAt = expires
	return existing, raw, nil
}

// InvitePreview is the public view of an invite token.
type InvitePreview struct {
	Email            string    `json:"email"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	ExpiresAt        time.Time `json:"expires_at"`
	UserExists       bool      `json:"user_exists"`
}

// PreviewInvite resolves a raw token for the accept UI.
func PreviewInvite(ctx context.Context, invites InviteRepository, users UserLookup, rawToken string) (*InvitePreview, error) {
	inv, orgName, err := invites.GetByTokenHash(ctx, HashInviteToken(rawToken))
	if err != nil {
		return nil, err
	}
	if inv.Status != InviteStatusPending {
		return nil, kernel.ErrNotFound
	}
	if timeNowUTC().After(inv.ExpiresAt) {
		return nil, kernel.ErrInvalidRequest
	}
	_, _, _, lookupErr := users.FindByEmail(ctx, inv.Email)
	exists := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, kernel.ErrNotFound) {
		return nil, lookupErr
	}
	return &InvitePreview{
		Email:            inv.Email,
		OrganizationID:   inv.OrganizationID,
		OrganizationName: orgName,
		ExpiresAt:        inv.ExpiresAt,
		UserExists:       exists,
	}, nil
}

// AcceptInviteInput is the accept payload for new users.
type AcceptInviteInput struct {
	Name         string
	PasswordHash string // bcrypt hash for new users; unused when user exists
}

// UserCreator creates platform users during invite accept.
type UserCreator interface {
	CreateUser(ctx context.Context, id, email, name, passwordHash string) (*identity.User, error)
}

// AcceptInvite consumes a pending invite and ensures org membership.
func AcceptInvite(
	ctx context.Context,
	orgs OrganizationRepository,
	invites InviteRepository,
	users UserLookup,
	creator UserCreator,
	rawToken string,
	in AcceptInviteInput,
	newUserID string,
) (*OrgMember, *identity.User, error) {
	inv, _, err := invites.GetByTokenHash(ctx, HashInviteToken(rawToken))
	if err != nil {
		return nil, nil, err
	}
	if inv.Status != InviteStatusPending {
		return nil, nil, kernel.ErrNotFound
	}
	if timeNowUTC().After(inv.ExpiresAt) {
		return nil, nil, kernel.ErrInvalidRequest
	}

	userID, name, userEmail, lookupErr := users.FindByEmail(ctx, inv.Email)
	var user *identity.User
	if errors.Is(lookupErr, kernel.ErrNotFound) {
		name = strings.TrimSpace(in.Name)
		if name == "" || strings.TrimSpace(in.PasswordHash) == "" {
			return nil, nil, fmt.Errorf("%w: name and password required", kernel.ErrInvalidRequest)
		}
		if newUserID == "" {
			return nil, nil, kernel.ErrInvalidRequest
		}
		u, err := creator.CreateUser(ctx, newUserID, inv.Email, name, in.PasswordHash)
		if err != nil {
			return nil, nil, err
		}
		user = u
		userID, name, userEmail = u.ID, u.Name, u.Email
	} else if lookupErr != nil {
		return nil, nil, lookupErr
	} else {
		user = &identity.User{ID: userID, Email: userEmail, Name: name}
	}

	if _, err := orgs.GetMemberRole(ctx, userID, inv.OrganizationID); err == nil {
		_ = invites.MarkAccepted(ctx, inv.ID, timeNowUTC())
		return &OrgMember{UserID: userID, Email: userEmail, Name: name, Role: OrgRoleMember}, user, nil
	} else if !errors.Is(err, kernel.ErrNotFound) {
		return nil, nil, err
	}
	role := inv.Role
	if role == "" {
		role = OrgRoleMember
	}
	if err := orgs.AddMember(ctx, inv.OrganizationID, userID, role); err != nil {
		return nil, nil, err
	}
	if err := invites.MarkAccepted(ctx, inv.ID, timeNowUTC()); err != nil {
		return nil, nil, err
	}
	return &OrgMember{UserID: userID, Email: userEmail, Name: name, Role: role}, user, nil
}
