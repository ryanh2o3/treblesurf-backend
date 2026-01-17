package dynamodb

import (
	"context"
	"fmt"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.APIKeyRepository = (*APIKeyRepo)(nil)

type APIKeyRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewAPIKeyRepo(client *dynamodb.DynamoDB, tableName string) *APIKeyRepo {
	return &APIKeyRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *APIKeyRepo) Create(ctx context.Context, key *model.APIKey) error {
	item, err := dynamodbattribute.MarshalMap(key)
	if err != nil {
		return fmt.Errorf("marshaling api key: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("creating api key: %w", err)
	}

	return nil
}

func (r *APIKeyRepo) GetByKey(ctx context.Context, keyValue string) (*model.APIKey, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("key_value = :key_value"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":key_value": {S: aws.String(keyValue)},
		},
		Limit: aws.Int64(1),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning api keys: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, model.ErrAPIKeyNotFound
	}

	var apiKey model.APIKey
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &apiKey); err != nil {
		return nil, fmt.Errorf("unmarshaling api key: %w", err)
	}

	return &apiKey, nil
}

func (r *APIKeyRepo) List(ctx context.Context) ([]*model.APIKey, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning api keys: %w", err)
	}

	keys := make([]*model.APIKey, 0, len(result.Items))
	for _, item := range result.Items {
		var key model.APIKey
		if err := dynamodbattribute.UnmarshalMap(item, &key); err != nil {
			return nil, fmt.Errorf("unmarshaling api key: %w", err)
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

func (r *APIKeyRepo) Revoke(ctx context.Context, keyID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key_id": {S: aws.String(keyID)},
		},
	}

	if _, err := r.client.DeleteItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("revoking api key: %w", err)
	}

	return nil
}
