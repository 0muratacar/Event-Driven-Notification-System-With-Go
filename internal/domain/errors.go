package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrDuplicate          = errors.New("duplicate entry")
	ErrIdempotencyConflict = errors.New("idempotency key already exists with different parameters")
	ErrInvalidStatus      = errors.New("invalid status transition")
	ErrCannotCancel       = errors.New("notification cannot be cancelled in current status")
	ErrBatchTooLarge      = errors.New("batch size exceeds maximum of 1000")
	ErrInvalidChannel     = errors.New("invalid channel")
	ErrInvalidPriority    = errors.New("invalid priority")
	ErrTemplateRender     = errors.New("template rendering failed")
	ErrRateLimited        = errors.New("rate limited")
	ErrProviderFailure    = errors.New("delivery provider failure")
	ErrValidation         = errors.New("validation error")
)
