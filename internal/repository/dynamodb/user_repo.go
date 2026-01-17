package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewUserRepo(client *dynamodb.DynamoDB, tableName string) *UserRepo {
	return &UserRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	if result.Item == nil {
		return nil, model.ErrUserNotFound
	}

	var user model.User
	if err := dynamodbattribute.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("unmarshaling user: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByUUID(ctx context.Context, uuid string) (*model.User, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("#uuid = :uuid"),
		ExpressionAttributeNames: map[string]*string{
			"#uuid": aws.String("uuid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uuid": {S: aws.String(uuid)},
		},
		Limit: aws.Int64(1),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning user by uuid: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, model.ErrUserNotFound
	}

	var user model.User
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &user); err != nil {
		return nil, fmt.Errorf("unmarshaling user: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("marshaling user: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(email)"),
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		if isConditionalCheckFailed(err) {
			return model.ErrUserAlreadyExists
		}
		return fmt.Errorf("creating user: %w", err)
	}

	return nil
}

func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("marshaling user: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, email string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
	}

	if _, err := r.client.DeleteItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	return nil
}

func (r *UserRepo) UpdateTheme(ctx context.Context, email, theme string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
		UpdateExpression: aws.String("SET theme = :theme"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":theme": {S: aws.String(theme)},
		},
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("updating theme: %w", err)
	}

	return nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, email string, at time.Time) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
		UpdateExpression: aws.String("SET last_login = :last_login"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":last_login": {S: aws.String(at.UTC().Format(time.RFC3339))},
		},
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("updating last login: %w", err)
	}

	return nil
}

func isConditionalCheckFailed(err error) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return awsErr.Code() == dynamodb.ErrCodeConditionalCheckFailedException
	}
	return false
}
