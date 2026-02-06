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

var _ repository.StreamRequestRepository = (*StreamRequestRepo)(nil)

type StreamRequestRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewStreamRequestRepo(client *dynamodb.DynamoDB, tableName string) *StreamRequestRepo {
	return &StreamRequestRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *StreamRequestRepo) Save(ctx context.Context, request *model.StreamRequest) error {
	item, err := dynamodbattribute.MarshalMap(streamRequestItemFromModel(request))
	if err != nil {
		return fmt.Errorf("marshaling stream request: %w", err)
	}

	_, err = r.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("saving stream request: %w", err)
	}

	return nil
}

func (r *StreamRequestRepo) GetBySpotID(ctx context.Context, spotID string) (*model.StreamRequest, error) {
	result, err := r.client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"spot_id": {S: aws.String(spotID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting stream request: %w", err)
	}

	if len(result.Item) == 0 {
		return nil, repository.ErrNotFound
	}

	var requestRecord streamRequestItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &requestRecord); err != nil {
		return nil, fmt.Errorf("unmarshaling stream request: %w", err)
	}

	return requestRecord.toModel(), nil
}
