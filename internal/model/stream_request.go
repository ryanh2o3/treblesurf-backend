package model

import "time"

type StreamRequest struct {
	RequestedAt time.Time `json:"requested_at"`
	SpotID      string    `json:"spot_id"`
	RequestedBy string    `json:"requested_by"`
	Expiration  int64     `json:"expiration"`
}
