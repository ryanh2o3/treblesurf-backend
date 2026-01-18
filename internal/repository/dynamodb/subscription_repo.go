package dynamodb

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var _ repository.SpotSubscriptionRepository = (*SpotSubscriptionRepo)(nil)

type SpotSubscriptionRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewSpotSubscriptionRepo(client *dynamodb.DynamoDB, tableName string) *SpotSubscriptionRepo {
	return &SpotSubscriptionRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *SpotSubscriptionRepo) Save(ctx context.Context, spotIdentifier, userID, connectionID string) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"spot_id": {
				S: aws.String(spotIdentifier),
			},
			"user_id": {
				S: aws.String(userID),
			},
			"subscribed_at": {
				S: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
			"connection_id": {
				S: aws.String(connectionID),
			},
		},
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("saving subscription: %w", err)
	}

	return nil
}

func (r *SpotSubscriptionRepo) GetSubscribersBySpot(ctx context.Context, spotIdentifier string) ([]string, error) {
	if spotIdentifier == "" {
		return nil, fmt.Errorf("spot identifier is required")
	}

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {S: aws.String(spotIdentifier)},
		},
	}

	unique := make(map[string]struct{})
	for {
		output, err := r.client.QueryWithContext(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("querying subscriptions: %w", err)
		}
		for _, item := range output.Items {
			if attr, ok := item["user_id"]; ok && attr.S != nil && *attr.S != "" {
				unique[*attr.S] = struct{}{}
			}
		}
		if len(output.LastEvaluatedKey) == 0 {
			break
		}
		input.ExclusiveStartKey = output.LastEvaluatedKey
	}

	subscribers := make([]string, 0, len(unique))
	for userID := range unique {
		subscribers = append(subscribers, userID)
	}

	return subscribers, nil
}
