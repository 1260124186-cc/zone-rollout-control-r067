package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"solo-0014-zone-rollout-control/internal/domain"
)

const requestLimit = 1 << 20

type successBody struct {
	Data any          `json:"data"`
	Meta responseMeta `json:"meta"`
}

type responseMeta struct {
	RequestID string `json:"request_id,omitempty"`
}

type failureBody struct {
	Error domain.DomainError `json:"error"`
}

func decode(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, requestLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.Validation("request body must contain valid JSON", map[string]any{
			"reason": err.Error(),
		})
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.Validation("request body must contain one JSON value", nil)
	}
	return nil
}

func sendData(w http.ResponseWriter, r *http.Request, status int, value any) {
	sendJSON(w, status, successBody{
		Data: value,
		Meta: responseMeta{RequestID: requestID(r.Context())},
	})
}

func sendError(w http.ResponseWriter, r *http.Request, err error) {
	var target *domain.DomainError
	if !errors.As(err, &target) {
		target = domain.NewError(domain.CodeInternal, "internal server error", nil)
	}
	sendJSON(w, statusCode(target.Code), failureBody{Error: *target})
}

func sendJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func statusCode(code domain.ErrorCode) int {
	switch code {
	case domain.CodeValidationFailed:
		return http.StatusBadRequest
	case domain.CodeNotFound:
		return http.StatusNotFound
	case domain.CodeInvalidState,
		domain.CodeVersionConflict,
		domain.CodeIdempotencyConflict,
		domain.CodeDuplicateRevision,
		domain.CodeNodeUnavailable,
		domain.CodeLeaseExpired,
		domain.CodeLeaseNotClaimed,
		domain.CodeConflict:
		return http.StatusConflict
	case domain.CodeNoProbes:
		return http.StatusUnprocessableEntity
	case domain.CodeStorageFailure, domain.CodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
