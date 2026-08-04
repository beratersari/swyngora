package domain

import "errors"

// Domain-level sentinel errors. Transport maps these to HTTP status codes.
var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrUpstream        = errors.New("upstream data source error")
	ErrRateLimited     = errors.New("rate limited by upstream")
)
