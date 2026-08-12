// Package repository defines interfaces for data persistence operations.
// All repositories follow the repository pattern, abstracting data access
// from business logic and enabling easy testing through mock implementations.
package repository

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
)

// UserRepository handles user data persistence.
// Implementations should handle user CRUD operations and related queries.
type UserRepository interface {
	// GetByEmail retrieves a user by their email address.
	// Returns ErrNotFound if the user doesn't exist.
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// GetByUUID retrieves a user by their unique identifier.
	// Returns ErrNotFound if the user doesn't exist.
	GetByUUID(ctx context.Context, uuid string) (*model.User, error)

	// GetByAppleID retrieves a user by their Apple Sign in with Apple subject (sub).
	// Returns ErrNotFound if the user doesn't exist.
	GetByAppleID(ctx context.Context, appleID string) (*model.User, error)

	// Create stores a new user in the database.
	// Returns ErrAlreadyExists if a user with the same email exists.
	Create(ctx context.Context, user *model.User) error

	// Update modifies an existing user's data.
	// Returns ErrNotFound if the user doesn't exist.
	Update(ctx context.Context, user *model.User) error

	// Delete removes a user by their email address.
	// Returns ErrNotFound if the user doesn't exist.
	Delete(ctx context.Context, email string) error

	// UpdateTheme updates a user's theme preference.
	UpdateTheme(ctx context.Context, email, theme string) error

	// UpdateLastLogin updates the user's last login timestamp.
	UpdateLastLogin(ctx context.Context, email string, at time.Time) error
}

// ReportRepository handles surf report persistence.
// Surf reports are user-submitted observations about current surf conditions.
type ReportRepository interface {
	// Create stores a new surf report.
	Create(ctx context.Context, report *model.SurfReport) error

	// GetBySpot retrieves reports for a specific surf spot.
	// Results are ordered by timestamp descending (newest first).
	// Use limit=0 for no limit.
	GetBySpot(ctx context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error)

	// GetBySpotAndTimeRange retrieves reports within a specific time window.
	GetBySpotAndTimeRange(ctx context.Context, country, region, spot string, start, end time.Time) ([]*model.SurfReport, error)

	// ScanSince retrieves all reports since a given time across all spots.
	// Use limit=0 for no limit.
	ScanSince(ctx context.Context, since time.Time, limit int) ([]*model.SurfReport, error)
}

// LocationRepository handles location data persistence.
type LocationRepository interface {
	GetRegions(ctx context.Context, country string) ([]string, error)
	GetSpots(ctx context.Context, country, region string) ([]*model.LocationInfo, error)
	GetLocationInfo(ctx context.Context, country, region, spot string) (*model.LocationInfo, error)
	GetCoordinates(ctx context.Context, country, region, spot string) (lat, lon float64, err error)
}

// ForecastRepository handles forecast data persistence.
type ForecastRepository interface {
	// GetSpotForecast loads rows from the given DynamoDB source partitions (e.g. "stormglass", "imi_swan+weatherkit").
	// Pass multiple source strings to merge (e.g. for sources=all); a single string is the common case.
	// granularity must be "hourly" or "multiHour" (partition key suffix on spot_id).
	GetSpotForecast(
		ctx context.Context,
		country, region, spot string,
		partitionSources []string,
		granularity string,
	) ([]*model.Forecast, error)
	// GetCurrentConditions loads the earliest hourly row within a near-term window (used for /currentConditions).
	// partitionSources follows the same source selection rules as GetSpotForecast (e.g. preferred source vs default).
	GetCurrentConditions(
		ctx context.Context,
		country, region, spot string,
		partitionSources []string,
	) (*model.Forecast, error)
	GetForecastAtTime(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error)
}

// ForecastDataRepository handles raw forecast data queries needed for report logic.
type ForecastDataRepository interface {
	QuerySince(ctx context.Context, spotID string, since time.Time, limit int) ([]*model.ForecastDataPoint, error)
	QueryBetween(ctx context.Context, spotID string, start, end time.Time, limit int) ([]*model.ForecastDataPoint, error)
}

// BuoyDataRequest represents a request for buoy data at a specific time range.
type BuoyDataRequest struct {
	Start    time.Time
	End      time.Time
	BuoyName string
}

// BuoyRepository handles buoy data persistence.
// Buoy data includes wave height, period, direction, and other oceanographic measurements.
type BuoyRepository interface {
	// GetLiveData retrieves the most recent data point for a buoy.
	GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error)

	// GetDataAtTime retrieves buoy data closest to the specified time.
	GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error)

	// GetDataRange retrieves all data points within a time range.
	GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error)

	// GetBatchDataRanges retrieves data for multiple buoys in a single operation.
	// Returns a map of buoy name to data points.
	GetBatchDataRanges(ctx context.Context, requests []BuoyDataRequest) (map[string][]*model.BuoyData, error)

	// GetLocations retrieves location metadata for all buoys.
	GetLocations(ctx context.Context) (map[string]*model.BuoyLocation, error)
}

// SessionRepository handles session persistence.
type SessionRepository interface {
	Save(ctx context.Context, session *model.Session) error
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	Delete(ctx context.Context, sessionID string) error
	GetByUserID(ctx context.Context, userID string) ([]*model.Session, error)
}

// MediaRepository handles media file storage (S3).
// Used for storing and retrieving images and videos attached to surf reports.
type MediaRepository interface {
	// Upload stores a file with the given key and content type.
	Upload(ctx context.Context, key string, data []byte, contentType string) error

	// Download retrieves a file by its key.
	// Returns ErrNotFound if the file doesn't exist.
	Download(ctx context.Context, key string) ([]byte, error)

	// Exists checks whether a file exists for the given key.
	// Returns ErrNotFound if the file doesn't exist.
	Exists(ctx context.Context, key string) (bool, error)

	// Delete removes a file by its key.
	Delete(ctx context.Context, key string) error

	// GenerateUploadURL creates a presigned URL for direct client uploads.
	// The URL expires after the specified duration.
	GenerateUploadURL(ctx context.Context, key string, expires time.Duration) (string, error)

	// GenerateViewURL creates a presigned URL for viewing a file.
	// The URL expires after the specified duration.
	GenerateViewURL(ctx context.Context, key string, expires time.Duration) (string, error)
}

// APIKeyRepository handles API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKey(ctx context.Context, key string) (*model.APIKey, error)
	List(ctx context.Context) ([]*model.APIKey, error)
	Revoke(ctx context.Context, keyID string) error
}

// WebSocketRepository handles WebSocket connection persistence.
type WebSocketRepository interface {
	SaveConnection(ctx context.Context, conn *model.ConnectionInfo) error
	GetConnection(ctx context.Context, connectionID string) (*model.ConnectionInfo, error)
	DeleteConnection(ctx context.Context, connectionID string) error
	UpdateSpot(ctx context.Context, connectionID, spot string) error
	GetConnectionsByUserIDs(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error)
	UpdateLastActive(ctx context.Context, connectionID string) error
}

// SpotSubscriptionRepository handles spot subscription persistence.
type SpotSubscriptionRepository interface {
	Save(ctx context.Context, spotIdentifier, userID, connectionID string) error
	GetSubscribersBySpot(ctx context.Context, spotIdentifier string) ([]string, error)
}

// SwellPredictionRepository handles swell prediction persistence.
type SwellPredictionRepository interface {
	GetSpotPredictions(ctx context.Context, spotID string, start time.Time, limit int) ([]model.SwellPrediction, error)
	GetListSpotsPredictions(ctx context.Context, spotIDs []string, start time.Time, limit int) ([][]model.SwellPrediction, error)
	GetRegionPredictions(ctx context.Context, country, region string, start time.Time, perSpotLimit int) ([]model.SwellPrediction, error)
	GetSpotPredictionRange(ctx context.Context, spotID string, start, end time.Time) ([]model.SwellPrediction, error)
	GetRecentPredictions(ctx context.Context, cutoff time.Time, perSpotLimit int) ([]model.SwellPrediction, error)
	GetClosestPrediction(ctx context.Context, spotID string, now time.Time) (*model.SwellPrediction, error)
}

// StreamRequestRepository handles stream request persistence.
type StreamRequestRepository interface {
	Save(ctx context.Context, request *model.StreamRequest) error
	GetBySpotID(ctx context.Context, spotID string) (*model.StreamRequest, error)
}

// SnapshotRepository handles snapshot metadata persistence.
type SnapshotRepository interface {
	Save(ctx context.Context, snapshot *model.SpotSnapshot) error
	GetLatestBySpot(ctx context.Context, spotID string) (*model.SpotSnapshot, error)
}

// ContentReportRepository handles content report persistence.
// Content reports are user-submitted reports about surf reports that may violate guidelines.
type ContentReportRepository interface {
	// Create stores a new content report.
	Create(ctx context.Context, report *model.ContentReport) error

	// GetByID retrieves a content report by its ID.
	// Returns ErrNotFound if the report doesn't exist.
	GetByID(ctx context.Context, id string) (*model.ContentReport, error)

	// GetBySurfReportID retrieves all reports for a specific surf report.
	GetBySurfReportID(ctx context.Context, surfReportID string) ([]*model.ContentReport, error)

	// GetByReporterID retrieves all reports submitted by a specific user.
	GetByReporterID(ctx context.Context, userID string) ([]*model.ContentReport, error)

	// GetPendingReports retrieves pending reports for the moderation queue.
	// Results are ordered by created_at ascending (oldest first).
	GetPendingReports(ctx context.Context, limit, offset int) ([]*model.ContentReport, error)

	// UpdateStatus updates the status of a report.
	UpdateStatus(ctx context.Context, id, status, reviewedBy string) error

	// Resolve marks a report as resolved with the given resolution.
	Resolve(ctx context.Context, id, resolution, notes, reviewedBy string) error

	// CountByReporterSince counts reports submitted by a user since a given time.
	// Used for rate limiting.
	CountByReporterSince(ctx context.Context, userID string, since time.Time) (int, error)
}

// ModerationActionRepository handles moderation action persistence.
// Moderation actions are records of actions taken by moderators on reports.
type ModerationActionRepository interface {
	// Create stores a new moderation action.
	Create(ctx context.Context, action *model.ModerationAction) error

	// GetByReportID retrieves all actions for a specific report.
	GetByReportID(ctx context.Context, reportID string) ([]*model.ModerationAction, error)

	// List retrieves moderation actions with pagination.
	// Results are ordered by created_at descending (newest first).
	List(ctx context.Context, limit, offset int) ([]*model.ModerationAction, error)
}
