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

const userUUIDIndex = "user_uuid-index"

var _ repository.SpotAlertRepository = (*SpotAlertRepo)(nil)

type SpotAlertRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewSpotAlertRepo(client *dynamodb.DynamoDB, tableName string) *SpotAlertRepo {
	return &SpotAlertRepo{client: client, tableName: tableName}
}

func (r *SpotAlertRepo) Save(ctx context.Context, sub *model.SpotAlertSubscription) error {
	if sub == nil || sub.SpotID == "" || sub.UserUUID == "" {
		return fmt.Errorf("%w: spot id and user uuid are required", repository.ErrInvalidInput)
	}
	item, err := dynamodbattribute.MarshalMap(sub)
	if err != nil {
		return fmt.Errorf("marshaling spot alert: %w", err)
	}
	_, err = r.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("saving spot alert: %w", err)
	}
	return nil
}

func (r *SpotAlertRepo) Get(ctx context.Context, spotID, userUUID string) (*model.SpotAlertSubscription, error) {
	if spotID == "" || userUUID == "" {
		return nil, fmt.Errorf("%w: spot id and user uuid are required", repository.ErrInvalidInput)
	}
	out, err := r.client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       alertKey(spotID, userUUID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting spot alert: %w", err)
	}
	if out.Item == nil {
		return nil, repository.ErrNotFound
	}
	var sub model.SpotAlertSubscription
	if err := dynamodbattribute.UnmarshalMap(out.Item, &sub); err != nil {
		return nil, fmt.Errorf("unmarshaling spot alert: %w", err)
	}
	return &sub, nil
}

func (r *SpotAlertRepo) Delete(ctx context.Context, spotID, userUUID string) error {
	if spotID == "" || userUUID == "" {
		return fmt.Errorf("%w: spot id and user uuid are required", repository.ErrInvalidInput)
	}
	_, err := r.client.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key:       alertKey(spotID, userUUID),
	})
	if err != nil {
		return fmt.Errorf("deleting spot alert: %w", err)
	}
	return nil
}

func (r *SpotAlertRepo) GetByUser(ctx context.Context, userUUID string) ([]*model.SpotAlertSubscription, error) {
	if userUUID == "" {
		return nil, fmt.Errorf("%w: user uuid is required", repository.ErrInvalidInput)
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String(userUUIDIndex),
		KeyConditionExpression: aws.String("user_uuid = :uid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uid": {S: aws.String(userUUID)},
		},
	}
	return r.collectQuery(ctx, input)
}

func (r *SpotAlertRepo) GetBySpot(ctx context.Context, spotID string) ([]*model.SpotAlertSubscription, error) {
	if spotID == "" {
		return nil, fmt.Errorf("%w: spot id is required", repository.ErrInvalidInput)
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :sid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sid": {S: aws.String(spotID)},
		},
	}
	return r.collectQuery(ctx, input)
}

func (r *SpotAlertRepo) ListGoodSurfEnabled(ctx context.Context) ([]*model.SpotAlertSubscription, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("good_surf_enabled = :enabled"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":enabled": {BOOL: aws.Bool(true)},
		},
	}
	var subs []*model.SpotAlertSubscription
	for {
		out, err := r.client.ScanWithContext(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("scanning good-surf alerts: %w", err)
		}
		page, err := unmarshalAlerts(out.Items)
		if err != nil {
			return nil, err
		}
		subs = append(subs, page...)
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
	return subs, nil
}

func (r *SpotAlertRepo) DeleteByUser(ctx context.Context, userUUID string) error {
	subs, err := r.GetByUser(ctx, userUUID)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if err := r.Delete(ctx, sub.SpotID, userUUID); err != nil {
			return err
		}
	}
	return nil
}

func (r *SpotAlertRepo) UpdateLastNotifiedKey(ctx context.Context, spotID, userUUID, key string) error {
	if spotID == "" || userUUID == "" {
		return fmt.Errorf("%w: spot id and user uuid are required", repository.ErrInvalidInput)
	}
	_, err := r.client.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key:       alertKey(spotID, userUUID),
		UpdateExpression: aws.String(
			"SET last_good_surf_notified_key = :key",
		),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":key": {S: aws.String(key)},
		},
	})
	if err != nil {
		return fmt.Errorf("updating last notified key: %w", err)
	}
	return nil
}

func (r *SpotAlertRepo) collectQuery(ctx context.Context, input *dynamodb.QueryInput) ([]*model.SpotAlertSubscription, error) {
	var subs []*model.SpotAlertSubscription
	for {
		out, err := r.client.QueryWithContext(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("querying spot alerts: %w", err)
		}
		page, err := unmarshalAlerts(out.Items)
		if err != nil {
			return nil, err
		}
		subs = append(subs, page...)
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
	return subs, nil
}

func unmarshalAlerts(items []map[string]*dynamodb.AttributeValue) ([]*model.SpotAlertSubscription, error) {
	subs := make([]*model.SpotAlertSubscription, 0, len(items))
	for _, item := range items {
		var sub model.SpotAlertSubscription
		if err := dynamodbattribute.UnmarshalMap(item, &sub); err != nil {
			return nil, fmt.Errorf("unmarshaling spot alert: %w", err)
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

func alertKey(spotID, userUUID string) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"spot_id":   {S: aws.String(spotID)},
		"user_uuid": {S: aws.String(userUUID)},
	}
}
