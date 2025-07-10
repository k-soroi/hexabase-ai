package domain

import "errors"

// Domain errors - these are sentinel errors used for error checking with errors.Is()
var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrAuthStateNotFound is returned when an auth state is not found
	ErrAuthStateNotFound = errors.New("auth state not found")
)
