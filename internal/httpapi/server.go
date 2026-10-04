package httpapi

import (
	"log/slog"
	"net/http"

	"solo-0014-zone-rollout-control/internal/allocation"
	"solo-0014-zone-rollout-control/internal/assessment"
	"solo-0014-zone-rollout-control/internal/fleet"
	"solo-0014-zone-rollout-control/internal/plan"
	"solo-0014-zone-rollout-control/internal/platform"
	"solo-0014-zone-rollout-control/internal/probe"
)

type Server struct {
	plans       *plan.Service
	fleet       *fleet.Service
	allocations *allocation.Service
	probes      *probe.Service
	assessments *assessment.Service
	ids         platform.IDGenerator
	logger      *slog.Logger
}

func New(
	plans *plan.Service,
	fleetService *fleet.Service,
	allocations *allocation.Service,
	probes *probe.Service,
	assessments *assessment.Service,
	ids platform.IDGenerator,
	logger *slog.Logger,
) *Server {
	return &Server{
		plans:       plans,
		fleet:       fleetService,
		allocations: allocations,
		probes:      probes,
		assessments: assessments,
		ids:         ids,
		logger:      logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	return s.recoverer(s.withRequestID(s.withLogging(mux)))
}
