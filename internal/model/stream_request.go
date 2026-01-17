package model

import "time"

type StreamRequest struct {
	RequestedAt time.Time `json:"requested_at" dynamodbav:"requested_at"`
	SpotID      string    `json:"spot_id" dynamodbav:"spot_id"`
	RequestedBy string    `json:"requested_by" dynamodbav:"requested_by"`
	Expiration  int64     `json:"expiration" dynamodbav:"expiration"`
}
