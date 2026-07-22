package controlplane

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

func parseUsageFilter(r *http.Request) (UsageFilter, error) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := UsageFilter{
		Limit:        limit,
		ProjectID:    q.Get("project_id"),
		APIKeyID:     q.Get("api_key_id"),
		CredentialID: q.Get("credential_id"),
		Model:        q.Get("model"),
		Modality:     q.Get("modality"),
		GroupBy:      q.Get("group_by"),
		ExcludeBYOK:  q.Get("exclude_byok") == "1" || strings.EqualFold(q.Get("exclude_byok"), "true"),
		BYOKOnly:     q.Get("byok_only") == "1" || strings.EqualFold(q.Get("byok_only"), "true"),
	}
	if f.ExcludeBYOK && f.BYOKOnly {
		return f, fmt.Errorf("%w: exclude_byok and byok_only are mutually exclusive", kernel.ErrInvalidRequest)
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return f, fmt.Errorf("%w: invalid from", kernel.ErrInvalidRequest)
			}
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return f, fmt.Errorf("%w: invalid to", kernel.ErrInvalidRequest)
			}
		}
		f.To = &t
	}
	return f, nil
}

func (s *Server) handleListUsage(w http.ResponseWriter, r *http.Request) {
	f, err := parseUsageFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := s.app.ListUsage(r.Context(), r.PathValue("orgID"), f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []UsageEvent{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUsageSummary(w http.ResponseWriter, r *http.Request) {
	f, err := parseUsageFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if f.GroupBy == "" {
		f.GroupBy = "day"
	}
	list, err := s.app.SummarizeUsage(r.Context(), r.PathValue("orgID"), f)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []UsageSummaryBucket{}
	}
	writeJSON(w, http.StatusOK, list)
}
