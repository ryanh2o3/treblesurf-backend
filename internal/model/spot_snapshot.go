package model

import "time"

type SpotSnapshot struct {
	Timestamp  time.Time `json:"timestamp" dynamodbav:"timestamp"`
	UploadedAt time.Time `json:"uploaded_at" dynamodbav:"uploaded_at"`
	SpotID     string    `json:"spot_id" dynamodbav:"spot_id"`
	ImageKey   string    `json:"image_key" dynamodbav:"image_key"`
}
