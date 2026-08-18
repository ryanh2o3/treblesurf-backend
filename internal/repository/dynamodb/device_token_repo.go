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

var _ repository.DeviceTokenRepository = (*DeviceTokenRepo)(nil)

type DeviceTokenRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewDeviceTokenRepo(client *dynamodb.DynamoDB, tableName string) *DeviceTokenRepo {
	return &DeviceTokenRepo{client: client, tableName: tableName}
}

func (r *DeviceTokenRepo) Save(ctx context.Context, token *model.DeviceToken) error {
	if token == nil || token.UserUUID == "" || token.Token == "" {
		return fmt.Errorf("%w: user uuid and token are required", repository.ErrInvalidInput)
	}
	item, err := dynamodbattribute.MarshalMap(token)
	if err != nil {
		return fmt.Errorf("marshaling device token: %w", err)
	}
	_, err = r.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("saving device token: %w", err)
	}
	return nil
}

func (r *DeviceTokenRepo) Delete(ctx context.Context, userUUID, token string) error {
	if userUUID == "" || token == "" {
		return fmt.Errorf("%w: user uuid and token are required", repository.ErrInvalidInput)
	}
	_, err := r.client.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"user_uuid": {S: aws.String(userUUID)},
			"token":     {S: aws.String(token)},
		},
	})
	if err != nil {
		return fmt.Errorf("deleting device token: %w", err)
	}
	return nil
}

func (r *DeviceTokenRepo) GetByUser(ctx context.Context, userUUID string) ([]*model.DeviceToken, error) {
	if userUUID == "" {
		return nil, fmt.Errorf("%w: user uuid is required", repository.ErrInvalidInput)
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("user_uuid = :uid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uid": {S: aws.String(userUUID)},
		},
	}
	return r.collectTokens(ctx, input)
}

func (r *DeviceTokenRepo) DeleteByUser(ctx context.Context, userUUID string) error {
	tokens, err := r.GetByUser(ctx, userUUID)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if token == nil {
			continue
		}
		if err := r.Delete(ctx, userUUID, token.Token); err != nil {
			return err
		}
	}
	return nil
}

func (r *DeviceTokenRepo) collectTokens(ctx context.Context, input *dynamodb.QueryInput) ([]*model.DeviceToken, error) {
	var tokens []*model.DeviceToken
	for {
		out, err := r.client.QueryWithContext(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("querying device tokens: %w", err)
		}
		for _, item := range out.Items {
			var token model.DeviceToken
			if err := dynamodbattribute.UnmarshalMap(item, &token); err != nil {
				return nil, fmt.Errorf("unmarshaling device token: %w", err)
			}
			tokens = append(tokens, &token)
		}
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
	return tokens, nil
}
