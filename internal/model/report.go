package model

import "time"

type SurfReport struct {
	ID              string    `json:"id" dynamodbav:"id"`
	UserEmail      string    `json:"user_email" dynamodbav:"user_email"`
	Country        string    `json:"country" dynamodbav:"country"`
	Region         string    `json:"region" dynamodbav:"region"`
	Spot           string    `json:"spot" dynamodbav:"spot"`
	Timestamp      time.Time `json:"timestamp" dynamodbav:"timestamp"`
	SwellSize      string    `json:"swell_size" dynamodbav:"swell_size"`
	WindAmount     string    `json:"wind_amount" dynamodbav:"wind_amount"`
	WindDirection  string    `json:"wind_direction" dynamodbav:"wind_direction"`
	SurfConditions string    `json:"surf_conditions" dynamodbav:"surf_conditions"`
	SurfDifficulty string    `json:"surf_difficulty" dynamodbav:"surf_difficulty"`
	ImageKey       string    `json:"image_key,omitempty" dynamodbav:"image_key,omitempty"`
	Notes          string    `json:"notes,omitempty" dynamodbav:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

type ReportImage struct {
	Key         string    `json:"key" dynamodbav:"key"`
	ReportID    string    `json:"report_id" dynamodbav:"report_id"`
	ImageData   []byte    `json:"image_data" dynamodbav:"image_data"`
	ContentType string    `json:"content_type" dynamodbav:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at" dynamodbav:"uploaded_at"`
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
	ImageData     string `json:"imageData"` // Base64 encoded image
	Date 		string `json:"date"`
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
	ImageKey      string `json:"imageKey"` // S3 key for pre-uploaded image
	Date          string `json:"date"`
}

type ReportWithIOSValidation struct {
	Country       string `json:"country"`
	Region        string `json:"region"`
	Spot          string `json:"spot"`
	SurfSize      string `json:"surfSize"`
	WindAmount    string `json:"windAmount"`
	WindDirection string `json:"windDirection"`
	Consistency   string `json:"consistency"`
	Quality       string `json:"quality"`
	Messiness     string `json:"messiness"`
	ImageKey      string `json:"imageKey,omitempty"` // S3 key for image (optional)
	VideoKey      string `json:"videoKey,omitempty"` // S3 key for video (optional)
	IOSValidated  bool   `json:"iosValidated"`       // Flag indicating iOS validation
	Date          string `json:"date"`
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
