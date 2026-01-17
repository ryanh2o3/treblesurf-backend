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

var _ repository.SnapshotRepository = (*SnapshotRepo)(nil)

type SnapshotRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewSnapshotRepo(client *dynamodb.DynamoDB, tableName string) *SnapshotRepo {
	return &SnapshotRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *SnapshotRepo) Save(ctx context.Context, snapshot *model.SpotSnapshot) error {
	item, err := dynamodbattribute.MarshalMap(snapshot)
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}

	_, err = r.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}

	return nil
}

func (r *SnapshotRepo) GetLatestBySpot(ctx context.Context, spotID string) (*model.SpotSnapshot, error) {
	result, err := r.client.QueryWithContext(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spotId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spotId": {S: aws.String(spotID)},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int64(1),
	})
	if err != nil {
		return nil, fmt.Errorf("querying snapshots: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var snapshot model.SpotSnapshot
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshaling snapshot: %w", err)
	}

	return &snapshot, nil
}
