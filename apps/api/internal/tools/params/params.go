// Package params validates query parameters into typed values with field-specific errors.
package params

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
)

// Q wraps url.Values and collects the first validation error.
type Q struct {
	v   url.Values
	err error
}

// New wraps q.
func New(q url.Values) *Q { return &Q{v: q} }

// Err returns the first validation error (an *httpapi.APIError) or nil.
func (q *Q) Err() error { return q.err }

func (q *Q) fail(field, msg string) {
	if q.err == nil {
		q.err = httpapi.InvalidInput(field, msg)
	}
}

// String returns a trimmed value (or def); required fails on empty; maxLen bounds length.
func (q *Q) String(name, def string, required bool, maxLen int) string {
	v := strings.TrimSpace(q.v.Get(name))
	if v == "" {
		if required {
			q.fail(name, fmt.Sprintf("%s is required", name))
		}
		return def
	}
	if len(v) > maxLen {
		q.fail(name, fmt.Sprintf("%s is too long (max %d)", name, maxLen))
		return def
	}
	return v
}

// Enum returns a value from allowed (case-insensitive match, canonical spelling returned).
func (q *Q) Enum(name, def string, allowed ...string) string {
	v := strings.TrimSpace(q.v.Get(name))
	if v == "" {
		return def
	}
	i := slices.IndexFunc(allowed, func(a string) bool { return strings.EqualFold(a, v) })
	if i < 0 {
		q.fail(name, fmt.Sprintf("%s must be one of %s", name, strings.Join(allowed, ", ")))
		return def
	}
	return allowed[i]
}

// Int parses an integer and rejects values outside [lo, hi] (no silent clamping).
func (q *Q) Int(name string, def, lo, hi int) int {
	v := strings.TrimSpace(q.v.Get(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < lo || n > hi {
		q.fail(name, fmt.Sprintf("%s must be an integer from %d to %d", name, lo, hi))
		return def
	}
	return n
}

// Bool parses true/false/1/0.
func (q *Q) Bool(name string, def bool) bool {
	v := strings.TrimSpace(q.v.Get(name))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		q.fail(name, name+" must be true or false")
		return def
	}
	return b
}

// Family parses family=4|6 (0 or absent = any).
func (q *Q) Family() int {
	switch q.Enum("family", "0", "0", "4", "6") {
	case "4":
		return 4
	case "6":
		return 6
	}
	return 0
}
