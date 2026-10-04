package domain

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeValidationFailed    ErrorCode = "validation_failed"
	CodeNotFound            ErrorCode = "not_found"
	CodeInvalidState        ErrorCode = "invalid_state"
	CodeVersionConflict     ErrorCode = "version_conflict"
	CodeIdempotencyConflict ErrorCode = "idempotency_conflict"
	CodeDuplicateRevision   ErrorCode = "duplicate_revision"
	CodeNodeUnavailable     ErrorCode = "node_unavailable"
	CodeLeaseExpired        ErrorCode = "lease_expired"
	CodeLeaseNotClaimed     ErrorCode = "lease_not_claimed"
	CodeNoProbes            ErrorCode = "no_probes"
	CodeConflict            ErrorCode = "conflict"
	CodeStorageFailure      ErrorCode = "storage_failure"
	CodeInternal            ErrorCode = "internal_error"
)

type DomainError struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *DomainError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code ErrorCode, message string, details map[string]any) *DomainError {
	return &DomainError{Code: code, Message: message, Details: details}
}

func ErrorCodeOf(err error) ErrorCode {
	var target *DomainError
	if errors.As(err, &target) {
		return target.Code
	}
	return CodeInternal
}

func NotFound(entity string, id string) *DomainError {
	return NewError(CodeNotFound, entity+" was not found", map[string]any{
		"entity": entity,
		"id":     id,
	})
}

func Validation(message string, details map[string]any) *DomainError {
	return NewError(CodeValidationFailed, message, details)
}

func InvalidState(message string, details map[string]any) *DomainError {
	return NewError(CodeInvalidState, message, details)
}
