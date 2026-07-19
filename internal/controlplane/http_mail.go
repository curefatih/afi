package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/mail"
)

func (s *Server) handleGetOrgMail(w http.ResponseWriter, r *http.Request) {
	org, err := s.api.GetOrganization(r.Context(), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"selected":          org.MailProvider,
		"default_provider":  s.cfg.Mail.DefaultProvider,
		"enabled_providers": enabledMailProviders(s.cfg),
		"from":              s.cfg.Mail.From,
		"public_app_url":    s.cfg.Mail.PublicAppURL,
	})
}

func (s *Server) handleUpdateOrgMail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider *string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Provider == nil {
		writeErr(w, http.StatusBadRequest, "provider required")
		return
	}
	provider := mail.ProviderName(*body.Provider)
	if provider != "" {
		enabled := false
		for _, p := range enabledMailProviders(s.cfg) {
			if p == provider {
				enabled = true
				break
			}
		}
		if !enabled {
			writeErr(w, http.StatusBadRequest, "provider is not enabled on this deployment")
			return
		}
	}
	org, err := s.api.SetOrgMailProvider(r.Context(), r.PathValue("orgID"), provider)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invalid provider")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"selected":          org.MailProvider,
		"default_provider":  s.cfg.Mail.DefaultProvider,
		"enabled_providers": enabledMailProviders(s.cfg),
	})
}

func (s *Server) handleTestOrgMail(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	user, err := s.api.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	org, err := s.api.GetOrganization(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	sender, _, err := resolveMailSender(s.cfg, org.MailProvider, s.log)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	msg := mail.TestMessage()
	msg.To = user.Email
	if err := sender.Send(r.Context(), msg); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "to": user.Email})
}

func (s *Server) sendInviteMail(ctx context.Context, orgID string, outcome *InviteOutcome, rawToken string) error {
	if outcome == nil {
		return nil
	}
	org, err := s.api.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}
	sender, _, err := resolveMailSender(s.cfg, org.MailProvider, s.log)
	if err != nil {
		return err
	}
	base := strings.TrimRight(s.cfg.Mail.PublicAppURL, "/")
	var msg mail.Message
	switch outcome.Status {
	case "added":
		if outcome.Member == nil {
			return nil
		}
		msg = mail.InviteExistingUser(org.Name, base+"/auth/login")
		msg.To = outcome.Member.Email
	case "invited":
		if outcome.Invite == nil || rawToken == "" {
			return nil
		}
		msg = mail.InviteNewUser(org.Name, base+"/auth/invite/"+rawToken)
		msg.To = outcome.Invite.Email
	default:
		return nil
	}
	return sender.Send(ctx, msg)
}
