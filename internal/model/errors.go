package model

import "errors"

// Custom error types for surf report operations
var (
	ErrImageNotSurfRelated = errors.New("image does not appear to be surf-related")
	ErrImageAnalysisFailed = errors.New("image analysis failed")
	ErrImageUploadFailed   = errors.New("image upload failed")
	ErrInvalidImageData    = errors.New("invalid image data")
	ErrImageValidationFailed = errors.New("image validation failed")
	ErrImageRetrievalFailed = errors.New("failed to retrieve pre-uploaded image")
)

// ImageValidationError wraps image validation errors with additional context
type ImageValidationError struct {
	Err     error
	Message string
}

func (e *ImageValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *ImageValidationError) Unwrap() error {
	return e.Err
}

// NewImageValidationError creates a new image validation error
func NewImageValidationError(err error, message string) *ImageValidationError {
	return &ImageValidationError{
		Err:     err,
		Message: message,
	}
}
