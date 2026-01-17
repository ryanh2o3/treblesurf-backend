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
	GetSpotForecast(ctx context.Context, country, region, spot string) ([]*model.Forecast, error)
	GetCurrentConditions(ctx context.Context, country, region, spot string) (*model.Forecast, error)
	GetForecastAtTime(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error)
	GetRegionForecast(ctx context.Context, country, region string, forecastDate time.Time) ([]*model.Forecast, error)
}

// ForecastDataRepository handles raw forecast data queries needed for report logic.
type ForecastDataRepository interface {
	QuerySince(ctx context.Context, spotID string, since time.Time, limit int) ([]map[string]interface{}, error)
	QueryBetween(ctx context.Context, spotID string, start, end time.Time, limit int) ([]map[string]interface{}, error)
}

// BuoyDataRequest represents a request for buoy data at a specific time range.
type BuoyDataRequest struct {
	BuoyName string
	Start    time.Time
	End      time.Time
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
}

// SwellPredictionRepository handles swell prediction persistence.
type SwellPredictionRepository interface {
	GetSpotPredictions(ctx context.Context, spotID string, start time.Time, limit int) ([]map[string]interface{}, error)
	GetListSpotsPredictions(ctx context.Context, spotIDs []string, start time.Time, limit int) ([][]map[string]interface{}, error)
	GetRegionPredictions(ctx context.Context, country, region string, start time.Time, perSpotLimit int) ([]map[string]interface{}, error)
	GetSpotPredictionRange(ctx context.Context, spotID string, start, end time.Time) ([]map[string]interface{}, error)
	GetRecentPredictions(ctx context.Context, cutoff time.Time, perSpotLimit int) ([]map[string]interface{}, error)
	GetClosestPrediction(ctx context.Context, spotID string, now time.Time) (map[string]interface{}, error)
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
