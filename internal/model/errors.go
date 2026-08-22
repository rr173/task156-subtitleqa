package model

import "errors"

// Domain errors used across packages. The HTTP layer maps them to status codes.
var (
	// ErrValidation signals a malformed or business-invalid request.
	ErrValidation = errors.New("validation error")
	// ErrNotFound signals a referenced entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict signals a concurrent edit that was rejected.
	ErrConflict = errors.New("edit conflict")
	// ErrPublished signals the operation is forbidden on a published entity.
	ErrPublished = errors.New("already published")
	// ErrWithdrawn signals the operation is forbidden on a withdrawn version.
	ErrWithdrawn = errors.New("version withdrawn")
)

// FieldError carries a field-level validation message and unwraps to
// ErrValidation so callers can detect validation failures uniformly.
type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string { return e.Field + ": " + e.Message }

func (e FieldError) Unwrap() error { return ErrValidation }
