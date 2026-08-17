package service

import "errors"

// Report and surf-report media errors for HTTP mapping via errors.Is.
var (
	ErrSurfReportsQuery = errors.New("failed to query surf reports")

	ErrReportPresignedUploadURL = errors.New("failed to generate presigned upload URL")
	ErrReadReportImage          = errors.New("failed to read report image data")
	ErrReadReportVideo          = errors.New("failed to read report video data")

	ErrReportVideoKeyRequired = errors.New("video key is required")

	ErrReportUserNotFound     = errors.New("user not found")
	ErrReportUserMissingUUID  = errors.New("user does not have a UUID")
	ErrReportUserLookupFailed = errors.New("failed to get user")

	ErrReportVideoNotFound     = errors.New("video not found or not accessible")
	ErrReportVideoAccessDenied = errors.New("access denied")
	ErrReportPresignedViewURL  = errors.New("failed to generate presigned view URL")

	ErrReportMediaKeyRequired = errors.New("media key is required")
	ErrReportMediaDelete      = errors.New("failed to delete report media from storage")

	ErrReportWebSocketUnavailable = errors.New("websocket service not initialized")
)
