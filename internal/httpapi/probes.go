package httpapi

import (
	"net/http"

	"solo-0014-zone-rollout-control/internal/domain"
)

func (s *Server) submitProbe(w http.ResponseWriter, r *http.Request) {
	var input domain.ProbeInput
	if err := decode(w, r, &input); err != nil {
		sendError(w, r, err)
		return
	}
	result, err := s.probes.Submit(r.Context(), r.PathValue("id"), input)
	if err != nil {
		sendError(w, r, err)
		return
	}
	sendData(w, r, http.StatusCreated, result)
}
