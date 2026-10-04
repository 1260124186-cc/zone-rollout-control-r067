package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"solo-0014-zone-rollout-control/internal/domain"
)

type versionInput struct {
	ExpectedVersion uint64 `json:"expected_version"`
}

func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) {
	var input domain.PlanCreateInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	plan, err := s.plans.Create(r.Context(), input)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusCreated, plan)
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			sendError(w, r, domain.Validation("limit must be an integer", map[string]any{"limit": raw}))
			return
		}
		limit = parsed
	}
	state := domain.PlanState(strings.TrimSpace(r.URL.Query().Get("state")))
	if state != "" && state != domain.PlanDraft && state != domain.PlanActive && state != domain.PlanCompleted {
		sendError(w, r, domain.Validation("plan state is not supported", map[string]any{"state": state}))
		return
	}
	page, err := s.plans.List(r.Context(), state, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusOK, page)
}

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	result, err := s.plans.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusOK, result)
}

func (s *Server) addRevision(w http.ResponseWriter, r *http.Request) {
	var input domain.RevisionCreateInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	result, err := s.plans.AddRevision(r.Context(), r.PathValue("id"), input)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusCreated, result)
}

func (s *Server) activatePlan(w http.ResponseWriter, r *http.Request) {
	var input versionInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	if input.ExpectedVersion == 0 {
		sendError(w, r, domain.Validation("expected_version must be greater than zero", nil))
		return
	}
	result, err := s.plans.Activate(r.Context(), r.PathValue("id"), input.ExpectedVersion)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusOK, result)
}

func (s *Server) completePlan(w http.ResponseWriter, r *http.Request) {
	var input versionInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	if input.ExpectedVersion == 0 {
		sendError(w, r, domain.Validation("expected_version must be greater than zero", nil))
		return
	}
	result, err := s.plans.Complete(r.Context(), r.PathValue("id"), input.ExpectedVersion)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusOK, result)
}
