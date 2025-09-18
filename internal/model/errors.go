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
	
	// Video-related errors
	ErrVideoUploadFailed     = errors.New("video upload failed")
	ErrVideoRetrievalFailed  = errors.New("failed to retrieve video")
	ErrInvalidVideoFormat    = errors.New("invalid video format")
	ErrVideoTooLarge         = errors.New("video file too large")
	ErrIOSValidationRequired = errors.New("iOS validation required for this endpoint")
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
