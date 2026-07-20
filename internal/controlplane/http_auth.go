package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if s.auth == nil {
		writeErr(w, http.StatusInternalServerError, "auth not configured")
		return
	}
	tok, err := s.auth.LoginWithPassword(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, identity.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	user, err := s.api.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role,
	})
}

func (s *Server) handleListSSOProviders(w http.ResponseWriter, _ *http.Request) {
	var providers []platform.SSOProviderInfo
	if s.auth != nil {
		providers = s.auth.ListSSOProviders()
	}
	if providers == nil {
		providers = []platform.SSOProviderInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (s *Server) handleSSOStart(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("provider")
	if s.auth == nil {
		writeErr(w, http.StatusServiceUnavailable, "sso not configured")
		return
	}
	returnTo := r.URL.Query().Get("redirect")
	authURL, err := s.auth.BeginSSO(providerID, returnTo)
	if err != nil {
		status, msg := mapSSOError(err)
		writeErr(w, status, msg)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("provider")
	if s.auth == nil {
		http.Redirect(w, r, s.appSSOErrorURL("sso not configured"), http.StatusFound)
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		desc := r.URL.Query().Get("error_description")
		if desc == "" {
			desc = errParam
		}
		http.Redirect(w, r, s.auth.AppSSOCallbackURL("", "", desc), http.StatusFound)
		return
	}
	result, err := s.auth.CompleteSSO(r.Context(), providerID, code, state)
	if err != nil {
		_, msg := mapSSOError(err)
		http.Redirect(w, r, s.auth.AppSSOCallbackURL("", "", msg), http.StatusFound)
		return
	}
	http.Redirect(w, r, s.auth.AppSSOCallbackURL(result.Token, result.ReturnTo, ""), http.StatusFound)
}

func (s *Server) appSSOErrorURL(msg string) string {
	if s.auth != nil {
		return s.auth.AppSSOCallbackURL("", "", msg)
	}
	base := "http://localhost:3000"
	if s.cfg != nil && s.cfg.Mail.PublicAppURL != "" {
		base = s.cfg.Mail.PublicAppURL
	}
	u, err := url.Parse(base + "/auth/sso/callback")
	if err != nil {
		return base + "/auth/sso/callback?error=" + url.QueryEscape(msg)
	}
	q := u.Query()
	q.Set("error", msg)
	u.RawQuery = q.Encode()
	return u.String()
}

func mapSSOError(err error) (status int, msg string) {
	switch {
	case errors.Is(err, identity.ErrSSODisabled):
		return http.StatusNotFound, "sso disabled"
	case errors.Is(err, identity.ErrUnknownProvider):
		return http.StatusNotFound, "unknown sso provider"
	case errors.Is(err, identity.ErrInvalidSSOState):
		return http.StatusBadRequest, "invalid or expired sso state"
	case errors.Is(err, identity.ErrEmailNotVerified):
		return http.StatusUnauthorized, "email not verified"
	case errors.Is(err, identity.ErrMissingFederatedEmail):
		return http.StatusUnauthorized, "federated email required"
	case errors.Is(err, identity.ErrMissingFederatedSubject):
		return http.StatusUnauthorized, "federated subject required"
	case errors.Is(err, kernel.ErrInvalidRequest):
		return http.StatusBadRequest, "invalid request"
	default:
		return http.StatusBadGateway, err.Error()
	}
}
