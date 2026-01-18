package model

import "time"

type SurfReport struct {
	Timestamp         time.Time `json:"timestamp" dynamodbav:"timestamp"`
	UpdatedAt         time.Time `json:"updated_at" dynamodbav:"updated_at"`
	CreatedAt         time.Time `json:"created_at" dynamodbav:"created_at"`
	Notes             string    `json:"notes,omitempty" dynamodbav:"notes,omitempty"`
	UserEmail         string    `json:"user_email,omitempty" dynamodbav:"UserEmail,omitempty"`
	Region            string    `json:"region" dynamodbav:"region"`
	ID                string    `json:"id" dynamodbav:"id"`
	WindAmount        string    `json:"wind_amount,omitempty" dynamodbav:"WindAmount,omitempty"`
	WindDirection     string    `json:"wind_direction,omitempty" dynamodbav:"WindDirection,omitempty"`
	SurfConditions    string    `json:"surf_conditions" dynamodbav:"surf_conditions"`
	SurfDifficulty    string    `json:"surf_difficulty" dynamodbav:"surf_difficulty"`
	ImageKey          string    `json:"image_key,omitempty" dynamodbav:"ImageKey,omitempty"`
	SwellSize         string    `json:"swell_size,omitempty" dynamodbav:"swell_size,omitempty"`
	Country           string    `json:"country" dynamodbav:"country"`
	Spot              string    `json:"spot" dynamodbav:"spot"`
	VideoKey          string    `json:"video_key,omitempty" dynamodbav:"VideoKey,omitempty"`
	Reporter          string    `json:"reporter,omitempty" dynamodbav:"Reporter,omitempty"`
	ReportedBy        string    `json:"reported_by,omitempty" dynamodbav:"reportedBy,omitempty"`
	MediaType         string    `json:"media_type,omitempty" dynamodbav:"MediaType,omitempty"`
	CountryRegionSpot string    `json:"country_region_spot,omitempty" dynamodbav:"country_region_spot,omitempty"`
	SurfSize          string    `json:"surf_size,omitempty" dynamodbav:"SurfSize,omitempty"`
	Consistency       string    `json:"consistency,omitempty" dynamodbav:"Consistency,omitempty"`
	Quality           string    `json:"quality,omitempty" dynamodbav:"Quality,omitempty"`
	Messiness         string    `json:"messiness,omitempty" dynamodbav:"Messiness,omitempty"`
	Time              string    `json:"time,omitempty" dynamodbav:"Time,omitempty"`
	DateReported      string    `json:"date_reported,omitempty" dynamodbav:"dateReported,omitempty"`
	IOSValidated      bool      `json:"ios_validated,omitempty" dynamodbav:"IOSValidated,omitempty"`
}

type ReportImage struct {
	UploadedAt  time.Time `json:"uploaded_at" dynamodbav:"uploaded_at"`
	Key         string    `json:"key" dynamodbav:"key"`
	ReportID    string    `json:"report_id" dynamodbav:"report_id"`
	ContentType string    `json:"content_type" dynamodbav:"content_type"`
	ImageData   []byte    `json:"image_data" dynamodbav:"image_data"`
}

type ReportWithImage struct {
	Country       string `json:"country"`
	Region        string `json:"region"`
	Spot          string `json:"spot"`
	SurfSize      string `json:"surfSize"`
	WindAmount    string `json:"windAmount"`
	WindDirection string `json:"windDirection"`
	Consistency   string `json:"consistency"`
	Quality       string `json:"quality"`
	Messiness     string `json:"messiness"`
	ImageData     string `json:"imageData"`
	Date          string `json:"date"`
}

type ReportWithS3Image struct {
	Country       string `json:"country"`
	Region        string `json:"region"`
	Spot          string `json:"spot"`
	SurfSize      string `json:"surfSize"`
	WindAmount    string `json:"windAmount"`
	WindDirection string `json:"windDirection"`
	Consistency   string `json:"consistency"`
	Quality       string `json:"quality"`
	Messiness     string `json:"messiness"`
	ImageKey      string `json:"imageKey"`
	Date          string `json:"date"`
}

type ReportWithIOSValidation struct {
	Consistency   string `json:"consistency"`
	Region        string `json:"region"`
	Spot          string `json:"spot"`
	SurfSize      string `json:"surfSize"`
	WindAmount    string `json:"windAmount"`
	WindDirection string `json:"windDirection"`
	Country       string `json:"country"`
	Quality       string `json:"quality"`
	Messiness     string `json:"messiness"`
	ImageKey      string `json:"imageKey,omitempty"`
	VideoKey      string `json:"videoKey,omitempty"`
	Date          string `json:"date"`
	IOSValidated  bool   `json:"iosValidated"`
}

type PresignedUploadResponse struct {
	UploadURL string `json:"uploadUrl"`
	ImageKey  string `json:"imageKey"`
	ExpiresAt string `json:"expiresAt"`
}

type VideoUploadResponse struct {
	UploadURL string `json:"uploadUrl"`
	VideoKey  string `json:"videoKey"`
	ExpiresAt string `json:"expiresAt"`
}

type VideoResponse struct {
	VideoData   string `json:"videoData"`
	ContentType string `json:"contentType"`
}

type VideoViewURLResponse struct {
	ViewURL   string `json:"viewURL"`
	ExpiresAt string `json:"expiresAt"`
}
