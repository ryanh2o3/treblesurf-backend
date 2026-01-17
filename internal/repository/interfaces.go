package repository

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
)

// UserRepository handles user data persistence.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUUID(ctx context.Context, uuid string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, email string) error
	UpdateTheme(ctx context.Context, email, theme string) error
	UpdateLastLogin(ctx context.Context, email string, at time.Time) error
}

// ReportRepository handles surf report persistence.
type ReportRepository interface {
	Create(ctx context.Context, report *model.SurfReport) error
	GetBySpot(ctx context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error)
	GetBySpotAndTimeRange(ctx context.Context, country, region, spot string, start, end time.Time) ([]*model.SurfReport, error)
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
}

// BuoyRepository handles buoy data persistence.
type BuoyRepository interface {
	GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error)
	GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error)
	GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error)
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
type MediaRepository interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) error
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	GenerateUploadURL(ctx context.Context, key string, expires time.Duration) (string, error)
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
}
