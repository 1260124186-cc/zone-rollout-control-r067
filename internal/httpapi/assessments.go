package httpapi

import (
	"net/http"

	"solo-0014-zone-rollout-control/internal/domain"
)

func (s *Server) assessPlan(w http.ResponseWriter, r *http.Request) {
	var input domain.AssessmentInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	result, err := s.assessments.Evaluate(r.Context(), r.PathValue("id"), input)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusCreated, result)
}

func (s *Server) latestAssessment(w http.ResponseWriter, r *http.Request) {
	result, err := s.assessments.Latest(r.Context(), r.PathValue("id"))
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusOK, result)
}
