package dynamodb

import "github.com/aws/aws-sdk-go/service/dynamodb"

// Client wraps a DynamoDB client to simplify dependency injection.
type Client struct {
	DB *dynamodb.DynamoDB
}

func NewClient(db *dynamodb.DynamoDB) *Client {
	return &Client{DB: db}
}
