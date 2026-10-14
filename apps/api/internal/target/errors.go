package target

import (
	"fmt"
	"net/netip"
)

// Error codes match docs/API_SPEC.md §1.2.
const (
	CodeInvalid       = "INVALID_INPUT"
	CodeUnresolvable  = "TARGET_UNRESOLVABLE"
	CodePrivateDenied = "TARGET_PRIVATE_DENIED"
	CodeBlocked       = "TARGET_BLOCKED"
)

// Error is a client-safe target error; httpapi maps it to the envelope via APICode/APIMessage.
type Error struct {
	Code    string
	Message string
	Field   string
	Addr    netip.Addr
	Err     error
}

func (e *Error) Error() string {
	s := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Addr.IsValid() {
		s += " (" + e.Addr.String() + ")"
	}
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	return s
}

func (e *Error) Unwrap() error { return e.Err }

// APICode is the public error code.
func (e *Error) APICode() string { return e.Code }

// APIMessage is the public message (never includes resolver internals).
func (e *Error) APIMessage() string { return e.Message }

// APIDetails adds structured details for clients.
func (e *Error) APIDetails() map[string]any {
	d := map[string]any{}
	if e.Field != "" {
		d["field"] = e.Field
	}
	if e.Addr.IsValid() {
		d["address"] = e.Addr.String()
	}
	return d
}

func invalid(field, msg string) error {
	return &Error{Code: CodeInvalid, Message: msg, Field: field}
}
