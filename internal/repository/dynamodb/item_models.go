package dynamodb

import (
	"time"

	"treblesurf-backend/internal/model"
)

type userItem struct {
	UUID       string `dynamodbav:"uuid"`
	Email      string `dynamodbav:"email"`
	Name       string `dynamodbav:"name"`
	Picture    string `dynamodbav:"picture"`
	FamilyName string `dynamodbav:"family_name"`
	GivenName  string `dynamodbav:"given_name"`
	CreatedAt  string `dynamodbav:"created_at"`
	LastLogin  string `dynamodbav:"last_login"`
	Theme      string `dynamodbav:"theme"`
	Role       string `dynamodbav:"role,omitempty"`
}

func userItemFromModel(u *model.User) userItem {
	if u == nil {
		return userItem{}
	}
	return userItem{
		UUID:       u.UUID,
		Email:      u.Email,
		Name:       u.Name,
		Picture:    u.Picture,
		FamilyName: u.FamilyName,
		GivenName:  u.GivenName,
		CreatedAt:  u.CreatedAt,
		LastLogin:  u.LastLogin,
		Theme:      u.Theme,
		Role:       u.Role,
	}
}

func (u *userItem) toModel() *model.User {
	return &model.User{
		UUID:       u.UUID,
		Email:      u.Email,
		Name:       u.Name,
		Picture:    u.Picture,
		FamilyName: u.FamilyName,
		GivenName:  u.GivenName,
		CreatedAt:  u.CreatedAt,
		LastLogin:  u.LastLogin,
		Theme:      u.Theme,
		Role:       u.Role,
	}
}

type reportItem struct {
	Timestamp         time.Time `dynamodbav:"timestamp"`
	UpdatedAt         time.Time `dynamodbav:"updated_at"`
	CreatedAt         time.Time `dynamodbav:"created_at"`
	Notes             string    `dynamodbav:"notes,omitempty"`
	UserEmail         string    `dynamodbav:"UserEmail,omitempty"`
	Region            string    `dynamodbav:"region"`
	ID                string    `dynamodbav:"id"`
	WindAmount        string    `dynamodbav:"WindAmount,omitempty"`
	WindDirection     string    `dynamodbav:"WindDirection,omitempty"`
	SurfConditions    string    `dynamodbav:"surf_conditions"`
	SurfDifficulty    string    `dynamodbav:"surf_difficulty"`
	ImageKey          string    `dynamodbav:"ImageKey,omitempty"`
	SwellSize         string    `dynamodbav:"swell_size,omitempty"`
	Country           string    `dynamodbav:"country"`
	Spot              string    `dynamodbav:"spot"`
	VideoKey          string    `dynamodbav:"VideoKey,omitempty"`
	Reporter          string    `dynamodbav:"Reporter,omitempty"`
	ReportedBy        string    `dynamodbav:"reportedBy,omitempty"`
	MediaType         string    `dynamodbav:"MediaType,omitempty"`
	CountryRegionSpot string    `dynamodbav:"country_region_spot,omitempty"`
	SurfSize          string    `dynamodbav:"SurfSize,omitempty"`
	Consistency       string    `dynamodbav:"Consistency,omitempty"`
	Quality           string    `dynamodbav:"Quality,omitempty"`
	Messiness         string    `dynamodbav:"Messiness,omitempty"`
	Time              string    `dynamodbav:"Time,omitempty"`
	DateReported      string    `dynamodbav:"dateReported,omitempty"`
	IOSValidated      bool      `dynamodbav:"IOSValidated,omitempty"`
}

func reportItemFromModel(r *model.SurfReport) reportItem {
	if r == nil {
		return reportItem{}
	}
	return reportItem{
		Timestamp:         r.Timestamp,
		UpdatedAt:         r.UpdatedAt,
		CreatedAt:         r.CreatedAt,
		Notes:             r.Notes,
		UserEmail:         r.UserEmail,
		Region:            r.Region,
		ID:                r.ID,
		WindAmount:        r.WindAmount,
		WindDirection:     r.WindDirection,
		SurfConditions:    r.SurfConditions,
		SurfDifficulty:    r.SurfDifficulty,
		ImageKey:          r.ImageKey,
		SwellSize:         r.SwellSize,
		Country:           r.Country,
		Spot:              r.Spot,
		VideoKey:          r.VideoKey,
		Reporter:          r.Reporter,
		ReportedBy:        r.ReportedBy,
		MediaType:         r.MediaType,
		CountryRegionSpot: r.CountryRegionSpot,
		SurfSize:          r.SurfSize,
		Consistency:       r.Consistency,
		Quality:           r.Quality,
		Messiness:         r.Messiness,
		Time:              r.Time,
		DateReported:      r.DateReported,
		IOSValidated:      r.IOSValidated,
	}
}

func (r *reportItem) toModel() *model.SurfReport {
	return &model.SurfReport{
		Timestamp:         r.Timestamp,
		UpdatedAt:         r.UpdatedAt,
		CreatedAt:         r.CreatedAt,
		Notes:             r.Notes,
		UserEmail:         r.UserEmail,
		Region:            r.Region,
		ID:                r.ID,
		WindAmount:        r.WindAmount,
		WindDirection:     r.WindDirection,
		SurfConditions:    r.SurfConditions,
		SurfDifficulty:    r.SurfDifficulty,
		ImageKey:          r.ImageKey,
		SwellSize:         r.SwellSize,
		Country:           r.Country,
		Spot:              r.Spot,
		VideoKey:          r.VideoKey,
		Reporter:          r.Reporter,
		ReportedBy:        r.ReportedBy,
		MediaType:         r.MediaType,
		CountryRegionSpot: r.CountryRegionSpot,
		SurfSize:          r.SurfSize,
		Consistency:       r.Consistency,
		Quality:           r.Quality,
		Messiness:         r.Messiness,
		Time:              r.Time,
		DateReported:      r.DateReported,
		IOSValidated:      r.IOSValidated,
	}
}

type forecastItem struct {
	CountryRegionSpot string    `dynamodbav:"country_region_spot"`
	ForecastDate      string    `dynamodbav:"ForecastDate"`
	Date              time.Time `dynamodbav:"date"`
	Conditions        string    `dynamodbav:"conditions"`
	Country           string    `dynamodbav:"country"`
	Region            string    `dynamodbav:"region"`
	Spot              string    `dynamodbav:"spot"`
	DateForecastedFor string    `dynamodbav:"dateForecastedFor"`
	Location          string    `dynamodbav:"location"`
	Hour              int       `dynamodbav:"hour"`
	WindSpeed         float64   `dynamodbav:"wind_speed"`
	WindDirection     float64   `dynamodbav:"wind_direction"`
	WaveHeight        float64   `dynamodbav:"wave_height"`
	WavePeriod        float64   `dynamodbav:"wave_period"`
	MaxPeriod         float64   `dynamodbav:"max_period"`
	WaveDirection     float64   `dynamodbav:"wave_direction"`
	Temperature       float64   `dynamodbav:"temperature"`
}

func (f *forecastItem) toModel() *model.Forecast {
	return &model.Forecast{
		CountryRegionSpot: f.CountryRegionSpot,
		ForecastDate:      f.ForecastDate,
		Date:              f.Date,
		Conditions:        f.Conditions,
		Country:           f.Country,
		Region:            f.Region,
		Spot:              f.Spot,
		DateForecastedFor: f.DateForecastedFor,
		Location:          f.Location,
		Hour:              f.Hour,
		WindSpeed:         f.WindSpeed,
		WindDirection:     f.WindDirection,
		WaveHeight:        f.WaveHeight,
		WavePeriod:        f.WavePeriod,
		MaxPeriod:         f.MaxPeriod,
		WaveDirection:     f.WaveDirection,
		Temperature:       f.Temperature,
	}
}

type sessionItem struct {
	SessionID string    `dynamodbav:"session_id"`
	UserID    string    `dynamodbav:"user_id"`
	ExpiresAt time.Time `dynamodbav:"expires_at"`
	JSON      string    `dynamodbav:"json_data"`
	TTL       int64     `dynamodbav:"ttl"`
}

func sessionItemFromModel(s *model.Session) sessionItem {
	if s == nil {
		return sessionItem{}
	}
	return sessionItem{
		SessionID: s.SessionID,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		JSON:      s.JSON,
		TTL:       s.TTL,
	}
}

func (s *sessionItem) toModel() *model.Session {
	return &model.Session{
		SessionID: s.SessionID,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		JSON:      s.JSON,
		TTL:       s.TTL,
	}
}

type apiKeyItem struct {
	KeyID       string    `dynamodbav:"key_id"`
	KeyValue    string    `dynamodbav:"key_value"`
	Description string    `dynamodbav:"description"`
	CreatedBy   string    `dynamodbav:"created_by"`
	CreatedAt   time.Time `dynamodbav:"created_at"`
	ExpiresAt   time.Time `dynamodbav:"expires_at"`
	Scopes      []string  `dynamodbav:"scopes"`
}

func apiKeyItemFromModel(k *model.APIKey) apiKeyItem {
	if k == nil {
		return apiKeyItem{}
	}
	return apiKeyItem{
		KeyID:       k.KeyID,
		KeyValue:    k.KeyValue,
		Description: k.Description,
		CreatedBy:   k.CreatedBy,
		CreatedAt:   k.CreatedAt,
		ExpiresAt:   k.ExpiresAt,
		Scopes:      k.Scopes,
	}
}

func (k *apiKeyItem) toModel() *model.APIKey {
	return &model.APIKey{
		KeyID:       k.KeyID,
		KeyValue:    k.KeyValue,
		Description: k.Description,
		CreatedBy:   k.CreatedBy,
		CreatedAt:   k.CreatedAt,
		ExpiresAt:   k.ExpiresAt,
		Scopes:      k.Scopes,
	}
}

type streamRequestItem struct {
	RequestedAt time.Time `dynamodbav:"requested_at"`
	SpotID      string    `dynamodbav:"spot_id"`
	RequestedBy string    `dynamodbav:"requested_by"`
	Expiration  int64     `dynamodbav:"expiration"`
}

func streamRequestItemFromModel(r *model.StreamRequest) streamRequestItem {
	if r == nil {
		return streamRequestItem{}
	}
	return streamRequestItem{
		RequestedAt: r.RequestedAt,
		SpotID:      r.SpotID,
		RequestedBy: r.RequestedBy,
		Expiration:  r.Expiration,
	}
}

func (r *streamRequestItem) toModel() *model.StreamRequest {
	return &model.StreamRequest{
		RequestedAt: r.RequestedAt,
		SpotID:      r.SpotID,
		RequestedBy: r.RequestedBy,
		Expiration:  r.Expiration,
	}
}

type spotSnapshotItem struct {
	Timestamp  time.Time `dynamodbav:"timestamp"`
	UploadedAt time.Time `dynamodbav:"uploaded_at"`
	SpotID     string    `dynamodbav:"spot_id"`
	ImageKey   string    `dynamodbav:"image_key"`
}

func spotSnapshotItemFromModel(s *model.SpotSnapshot) spotSnapshotItem {
	if s == nil {
		return spotSnapshotItem{}
	}
	return spotSnapshotItem{
		Timestamp:  s.Timestamp,
		UploadedAt: s.UploadedAt,
		SpotID:     s.SpotID,
		ImageKey:   s.ImageKey,
	}
}

func (s *spotSnapshotItem) toModel() *model.SpotSnapshot {
	return &model.SpotSnapshot{
		Timestamp:  s.Timestamp,
		UploadedAt: s.UploadedAt,
		SpotID:     s.SpotID,
		ImageKey:   s.ImageKey,
	}
}

type buoyDataItem struct {
	Timestamp        time.Time `dynamodbav:"timestamp"`
	BuoyName         string    `dynamodbav:"buoy_name"`
	WaveHeight       float64   `dynamodbav:"wave_height"`
	WavePeriod       float64   `dynamodbav:"wave_period"`
	MaxPeriod        float64   `dynamodbav:"max_period"`
	WaveDirection    float64   `dynamodbav:"wave_direction"`
	WindSpeed        float64   `dynamodbav:"wind_speed"`
	WindDirection    float64   `dynamodbav:"wind_direction"`
	Temperature      float64   `dynamodbav:"temperature"`
	Pressure         float64   `dynamodbav:"pressure"`
	SprTp            float64   `dynamodbav:"SprTp"`
	ThTp             float64   `dynamodbav:"ThTp"`
	MaxHeight        float64   `dynamodbav:"MaxHeight"`
	Gust             float64   `dynamodbav:"Gust"`
	AirTemperature   float64   `dynamodbav:"AirTemperature"`
	DewPoint         float64   `dynamodbav:"DewPoint"`
	RelativeHumidity float64   `dynamodbav:"RelativeHumidity"`
	Salinity         float64   `dynamodbav:"salinity"`
}

func (b *buoyDataItem) toModel() *model.BuoyData {
	return &model.BuoyData{
		Timestamp:        b.Timestamp,
		BuoyName:         b.BuoyName,
		WaveHeight:       b.WaveHeight,
		WavePeriod:       b.WavePeriod,
		MaxPeriod:        b.MaxPeriod,
		WaveDirection:    b.WaveDirection,
		WindSpeed:        b.WindSpeed,
		WindDirection:    b.WindDirection,
		Temperature:      b.Temperature,
		Pressure:         b.Pressure,
		SprTp:            b.SprTp,
		ThTp:             b.ThTp,
		MaxHeight:        b.MaxHeight,
		Gust:             b.Gust,
		AirTemperature:   b.AirTemperature,
		DewPoint:         b.DewPoint,
		RelativeHumidity: b.RelativeHumidity,
		Salinity:         b.Salinity,
	}
}

type buoyLocationItem struct {
	Name      string  `dynamodbav:"name"`
	RegionBuoy string `dynamodbav:"region_buoy,omitempty"`
	Country   string  `dynamodbav:"country"`
	Region    string  `dynamodbav:"region"`
	Spot      string  `dynamodbav:"spot"`
	Latitude  float64 `dynamodbav:"latitude"`
	Longitude float64 `dynamodbav:"longitude"`
}

func (b *buoyLocationItem) toModel() *model.BuoyLocation {
	return &model.BuoyLocation{
		Name:      b.Name,
		RegionBuoy: b.RegionBuoy,
		Country:   b.Country,
		Region:    b.Region,
		Spot:      b.Spot,
		Latitude:  b.Latitude,
		Longitude: b.Longitude,
	}
}

type connectionItem struct {
	ConnectionID string    `dynamodbav:"connection_id"`
	UserID       string    `dynamodbav:"user_id"`
	ConnectedAt  time.Time `dynamodbav:"connected_at"`
	LastActive   time.Time `dynamodbav:"last_active"`
	UserAgent    string    `dynamodbav:"user_agent"`
	IPAddress    string    `dynamodbav:"ip_address"`
	CurrentSpot  string    `dynamodbav:"CurrentSpot,omitempty"`
	TTL          int64     `dynamodbav:"ttl"`
}

func connectionItemFromModel(c *model.ConnectionInfo) connectionItem {
	if c == nil {
		return connectionItem{}
	}
	return connectionItem{
		ConnectionID: c.ConnectionID,
		UserID:       c.UserID,
		ConnectedAt:  c.ConnectedAt,
		LastActive:   c.LastActive,
		UserAgent:    c.UserAgent,
		IPAddress:    c.IPAddress,
		CurrentSpot:  c.CurrentSpot,
		TTL:          c.TTL,
	}
}

func (c *connectionItem) toModel() *model.ConnectionInfo {
	return &model.ConnectionInfo{
		ConnectionID: c.ConnectionID,
		UserID:       c.UserID,
		ConnectedAt:  c.ConnectedAt,
		LastActive:   c.LastActive,
		UserAgent:    c.UserAgent,
		IPAddress:    c.IPAddress,
		CurrentSpot:  c.CurrentSpot,
		TTL:          c.TTL,
	}
}

type contentReportItem struct {
	ReviewedAt      time.Time `dynamodbav:"reviewed_at,omitempty"`
	CreatedAt       time.Time `dynamodbav:"created_at"`
	UpdatedAt       time.Time `dynamodbav:"updated_at"`
	ID              string    `dynamodbav:"id"`
	SurfReportID    string    `dynamodbav:"surf_report_id"`
	ReporterUserID  string    `dynamodbav:"reporter_user_id"`
	Reason          string    `dynamodbav:"reason"`
	Description     string    `dynamodbav:"description,omitempty"`
	Status          string    `dynamodbav:"status"`
	ReviewedBy      string    `dynamodbav:"reviewed_by,omitempty"`
	Resolution      string    `dynamodbav:"resolution,omitempty"`
	ResolutionNotes string    `dynamodbav:"resolution_notes,omitempty"`
}

func contentReportItemFromModel(r *model.ContentReport) contentReportItem {
	if r == nil {
		return contentReportItem{}
	}
	return contentReportItem{
		ID:              r.ID,
		SurfReportID:    r.SurfReportID,
		ReporterUserID:  r.ReporterUserID,
		Reason:          r.Reason,
		Description:     r.Description,
		Status:          r.Status,
		ReviewedBy:      r.ReviewedBy,
		ReviewedAt:      r.ReviewedAt,
		Resolution:      r.Resolution,
		ResolutionNotes: r.ResolutionNotes,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func (r *contentReportItem) toModel() *model.ContentReport {
	return &model.ContentReport{
		ID:              r.ID,
		SurfReportID:    r.SurfReportID,
		ReporterUserID:  r.ReporterUserID,
		Reason:          r.Reason,
		Description:     r.Description,
		Status:          r.Status,
		ReviewedBy:      r.ReviewedBy,
		ReviewedAt:      r.ReviewedAt,
		Resolution:      r.Resolution,
		ResolutionNotes: r.ResolutionNotes,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

type moderationActionItem struct {
	CreatedAt   time.Time `dynamodbav:"created_at"`
	ID          string    `dynamodbav:"id"`
	ReportID    string    `dynamodbav:"report_id"`
	ActionType  string    `dynamodbav:"action_type"`
	TargetType  string    `dynamodbav:"target_type"`
	TargetID    string    `dynamodbav:"target_id"`
	PerformedBy string    `dynamodbav:"performed_by"`
	Notes       string    `dynamodbav:"notes,omitempty"`
}

func moderationActionItemFromModel(a *model.ModerationAction) moderationActionItem {
	if a == nil {
		return moderationActionItem{}
	}
	return moderationActionItem{
		ID:          a.ID,
		ReportID:    a.ReportID,
		ActionType:  a.ActionType,
		TargetType:  a.TargetType,
		TargetID:    a.TargetID,
		PerformedBy: a.PerformedBy,
		Notes:       a.Notes,
		CreatedAt:   a.CreatedAt,
	}
}

func (a *moderationActionItem) toModel() *model.ModerationAction {
	return &model.ModerationAction{
		ID:          a.ID,
		ReportID:    a.ReportID,
		ActionType:  a.ActionType,
		TargetType:  a.TargetType,
		TargetID:    a.TargetID,
		PerformedBy: a.PerformedBy,
		Notes:       a.Notes,
		CreatedAt:   a.CreatedAt,
	}
}

