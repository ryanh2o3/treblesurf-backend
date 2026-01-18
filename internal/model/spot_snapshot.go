package model

import "time"

type SpotSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	UploadedAt time.Time `json:"uploaded_at"`
	SpotID     string    `json:"spot_id"`
	ImageKey   string    `json:"image_key"`
}
