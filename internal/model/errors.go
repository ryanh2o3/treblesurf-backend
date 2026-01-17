package model

import "errors"

// User errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidTheme      = errors.New("invalid theme")
)

// Auth errors
var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Report errors
var (
	ErrReportNotFound    = errors.New("report not found")
	ErrInvalidReportData = errors.New("invalid report data")
)

// Image errors
var (
	ErrImageNotSurfRelated    = errors.New("image does not appear to be surf-related")
	ErrImageAnalysisFailed    = errors.New("image analysis failed")
	ErrImageUploadFailed      = errors.New("image upload failed")
	ErrInvalidImageData       = errors.New("invalid image data")
	ErrImageValidationFailed  = errors.New("image validation failed")
	ErrImageRetrievalFailed   = errors.New("failed to retrieve pre-uploaded image")
)

// Video errors
var (
	ErrVideoUploadFailed     = errors.New("video upload failed")
	ErrVideoRetrievalFailed  = errors.New("failed to retrieve video")
	ErrInvalidVideoFormat    = errors.New("invalid video format")
	ErrVideoTooLarge         = errors.New("video file too large")
	ErrIOSValidationRequired = errors.New("iOS validation required for this endpoint")
)

// Location errors
var (
	ErrLocationNotFound = errors.New("location not found")
	ErrInvalidLocation  = errors.New("invalid location parameters")
)

// Media errors
var (
	ErrMediaNotFound     = errors.New("media not found")
	ErrInvalidMediaKey   = errors.New("invalid media key")
	ErrMediaAccessDenied = errors.New("media access denied")
)

// API Key errors
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")
	ErrAPIKeyInvalid  = errors.New("api key invalid")
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
