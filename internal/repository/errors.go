// Package repository provides data access interfaces and common error definitions.
package repository

import "errors"

// Common repository errors for consistent error handling across the application.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when attempting to create a resource that already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrQueryFailed is returned when a database query fails.
	ErrQueryFailed = errors.New("query failed")

	// ErrMarshalFailed is returned when data marshaling fails.
	ErrMarshalFailed = errors.New("marshal failed")

	// ErrUnmarshalFailed is returned when data unmarshaling fails.
	ErrUnmarshalFailed = errors.New("unmarshal failed")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrConnectionFailed is returned when a database connection fails.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")

	// ErrConflict is returned when there's a write conflict.
	ErrConflict = errors.New("write conflict")
)
