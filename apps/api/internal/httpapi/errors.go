package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/digibaro/net-tools/apps/api/internal/limits"
)

// Code is a stable, machine-readable error code (docs/API_SPEC.md §1.2).
type Code string

const (
	CodeInvalidInput        Code = "INVALID_INPUT"
	CodeTargetUnresolvable  Code = "TARGET_UNRESOLVABLE"
	CodeTargetPrivateDenied Code = "TARGET_PRIVATE_DENIED"
	CodeTargetBlocked       Code = "TARGET_BLOCKED"
	CodeToolDisabled        Code = "TOOL_DISABLED"
	CodeProviderKeyMissing  Code = "PROVIDER_KEY_MISSING"
	CodeTimeout             Code = "TIMEOUT"
	CodeRateLimited         Code = "RATE_LIMITED"
	CodeBusy                Code = "BUSY"
	CodeProviderError       Code = "PROVIDER_ERROR"
	CodeInternal            Code = "INTERNAL"
	CodeNotFound            Code = "NOT_FOUND"
	CodeMethodNotAllowed    Code = "METHOD_NOT_ALLOWED"
)

var codeStatus = map[Code]int{
	CodeInvalidInput:        http.StatusBadRequest,
	CodeTargetUnresolvable:  http.StatusBadRequest,
	CodeTargetPrivateDenied: http.StatusForbidden,
	CodeTargetBlocked:       http.StatusForbidden,
	CodeToolDisabled:        http.StatusForbidden,
	CodeProviderKeyMissing:  http.StatusForbidden,
	CodeTimeout:             http.StatusRequestTimeout,
	CodeRateLimited:         http.StatusTooManyRequests,
	CodeBusy:                http.StatusServiceUnavailable,
	CodeProviderError:       http.StatusBadGateway,
	CodeInternal:            http.StatusInternalServerError,
	CodeNotFound:            http.StatusNotFound,
	CodeMethodNotAllowed:    http.StatusMethodNotAllowed,
}

// Status returns the HTTP status for a code (500 for unknown codes).
func (c Code) Status() int {
	if s, ok := codeStatus[c]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// APIError is an error that is safe to return to clients.
type APIError struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	// Err is the underlying cause; logged, never serialized.
	Err error `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return e.Err }

// NewError builds an APIError.
func NewError(code Code, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// InvalidInput reports a validation failure on one field.
func InvalidInput(field, message string) *APIError {
	return &APIError{Code: CodeInvalidInput, Message: message, Details: map[string]any{"field": field}}
}

// Coder is implemented by errors from other packages (e.g. target.Error) that carry a
// client-safe code without importing httpapi.
type Coder interface {
	error
	APICode() string
	APIMessage() string
	APIDetails() map[string]any
}

// ToAPIError maps any error to a client-safe APIError; unknown errors become INTERNAL
// with a generic message so internals never leak.
func ToAPIError(err error) *APIError {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr
	}
	if c, ok := errors.AsType[Coder](err); ok {
		return &APIError{Code: Code(c.APICode()), Message: c.APIMessage(), Details: c.APIDetails(), Err: err}
	}
	if errors.Is(err, limits.ErrBusy) {
		return &APIError{Code: CodeBusy, Message: "the server is busy, try again shortly", Err: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &APIError{Code: CodeTimeout, Message: "the operation timed out", Err: err}
	}
	return &APIError{Code: CodeInternal, Message: "internal error", Err: err}
}
