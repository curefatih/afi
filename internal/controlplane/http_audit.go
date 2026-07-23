package controlplane

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/audit"
	"github.com/curefatih/afi/internal/kernel"
)

func parseAuditFilter(r *http.Request) (audit.Filter, error) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := audit.Filter{
		Limit: limit,
		Name:  strings.TrimSpace(q.Get("name")),
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

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	f, err := parseAuditFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := s.app.ListAudit(r.Context(), r.PathValue("orgID"), f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []audit.Record{}
	}
	writeJSON(w, http.StatusOK, list)
}
