package controlplane

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/kernel"
)

type ctxClaimsKey int

const claimsKey ctxClaimsKey = 1

func claimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.ensureApp()
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		claims, err := ParseToken(s.cfg.Auth.JWTSecret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		ctx = platform.WithActor(ctx, claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireInternal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := CheckInternalToken(s.cfg.Auth.InternalToken, r.Header.Get("X-AFI-Internal-Token")); err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberFromPath(pathKey, func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgOwnerFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberFromPath(pathKey, func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgOwner(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaProject(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaProject(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaQuota(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaQuota(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetQuotaOrgID(r.Context(), r.PathValue("quotaID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaPolicy(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetPolicyOrgID(r.Context(), r.PathValue("policyID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		ok, err = s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaWasmHook(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetWasmHookOrgID(r.Context(), r.PathValue("hookID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		ok, err = s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaMCPBackend(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetMCPBackendOrgID(r.Context(), r.PathValue("backendID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		ok, err = s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaA2AAgent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetA2AAgentOrgID(r.Context(), r.PathValue("agentID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		ok, err = s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaCredential(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetCredentialOrgID(r.Context(), r.PathValue("credentialID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaCredentialAssignment(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetCredentialAssignmentOrgID(r.Context(), r.PathValue("assignmentID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireTeamAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.PathValue("teamID")
		_, err := s.members.GetTeamOrgID(r.Context(), teamID)
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.CanAccessTeam(r.Context(), teamID, claims.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireTeamManager(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.PathValue("teamID")
		_, err := s.members.GetTeamOrgID(r.Context(), teamID)
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.CanManageTeam(r.Context(), teamID, claims.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireTeamRoleChanger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.PathValue("teamID")
		_, err := s.members.GetTeamOrgID(r.Context(), teamID)
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.CanChangeTeamRoles(r.Context(), teamID, claims.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "only org admins and team owners can change team roles")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaProject(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaEnvironment(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetEnvironmentOrgID(r.Context(), r.PathValue("environmentID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaEnvironment(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaEnvironment(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetEnvironmentOrgID(r.Context(), r.PathValue("environmentID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgMemberViaProvider(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProviderOrgID(r.Context(), r.PathValue("providerID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaProvider(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaProvider(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProviderOrgID(r.Context(), r.PathValue("providerID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgMemberViaRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetRouteOrgID(r.Context(), r.PathValue("routeID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgAdminViaRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaRoute(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetRouteOrgID(r.Context(), r.PathValue("routeID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgMemberViaQuota(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetQuotaOrgID(r.Context(), r.PathValue("quotaID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}
