package domain

import "errors"

var (
	// ErrInvalidEmail is returned when the email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrMissingEmail is returned when the email is not provided
	ErrMissingEmail = errors.New("email is required")
)
