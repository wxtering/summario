package models

import "errors"

var (
	// Input and URL validation errors
	ErrEmptySource       = errors.New("source cannot be empty")
	ErrInvalidURL        = errors.New("invalid or unextractable source url")
	ErrUnsupportedSource = errors.New("unsupported source type")

	// Upstream resource availability errors
	ErrNotFound   = errors.New("requested resource not found")
	ErrRestricted = errors.New("resource is private, restricted or unplayable")

	// Upstream provider rate limit and quota errors
	ErrRateLimitExceeded = errors.New("upstream provider rate limit exceeded")

	// Content extraction errors
	ErrNoContent      = errors.New("no extractable content found in source")
	ErrUpstreamFailed = errors.New("failed to fetch content from upstream provider")
)
