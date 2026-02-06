package dynamodb

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.SessionRepository = (*SessionRepo)(nil)

type SessionRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewSessionRepo(client *dynamodb.DynamoDB, tableName string) *SessionRepo {
	return &SessionRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *SessionRepo) Save(ctx context.Context, session *model.Session) error {
	if session.TTL == 0 && !session.ExpiresAt.IsZero() {
		session.TTL = session.ExpiresAt.Unix()
	}
	item, err := dynamodbattribute.MarshalMap(sessionItemFromModel(session))
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	return nil
}

func (r *SessionRepo) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"session_id": {S: aws.String(sessionID)},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}
	if result.Item == nil {
		return nil, repository.ErrNotFound
	}

	var sessionRecord sessionItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &sessionRecord); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	sessionModel := sessionRecord.toModel()
	if !sessionModel.ExpiresAt.IsZero() && time.Now().After(sessionModel.ExpiresAt) {
		return nil, model.ErrSessionExpired
	}

	return sessionModel, nil
}

func (r *SessionRepo) Delete(ctx context.Context, sessionID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"session_id": {S: aws.String(sessionID)},
		},
	}

	if _, err := r.client.DeleteItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	return nil
}

func (r *SessionRepo) GetByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uid": {S: aws.String(userID)},
		},
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning sessions: %w", err)
	}

	sessions := make([]*model.Session, 0, len(result.Items))
	for _, item := range result.Items {
		var sessionRecord sessionItem
		if err := dynamodbattribute.UnmarshalMap(item, &sessionRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling session: %w", err)
		}
		sessionModel := sessionRecord.toModel()
		if sessionModel.ExpiresAt.IsZero() || time.Now().Before(sessionModel.ExpiresAt) {
			sessions = append(sessions, sessionModel)
		}
	}

	return sessions, nil
}

func (r *SessionRepo) EnsureSessionsTable(ctx context.Context) error {
	_, err := r.client.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	})
	if err == nil {
		return nil
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		input := &dynamodb.CreateTableInput{
			TableName: aws.String(r.tableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("session_id"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("session_id"),
					KeyType:       aws.String("HASH"),
				},
			},
			BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		}

		if _, createErr := r.client.CreateTableWithContext(ctx, input); createErr != nil {
			return fmt.Errorf("creating sessions table: %w", createErr)
		}

		if waitErr := r.client.WaitUntilTableExistsWithContext(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(r.tableName),
		}); waitErr != nil {
			return fmt.Errorf("waiting for sessions table: %w", waitErr)
		}
		return nil
	}

	return fmt.Errorf("describing sessions table: %w", err)
}

func (r *SessionRepo) EnableTTL(ctx context.Context) error {
	describe, err := r.client.DescribeTimeToLiveWithContext(ctx, &dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(r.tableName),
	})
	if err != nil {
		return fmt.Errorf("describing TTL: %w", err)
	}

	if describe.TimeToLiveDescription != nil {
		status := aws.StringValue(describe.TimeToLiveDescription.TimeToLiveStatus)
		if status == dynamodb.TimeToLiveStatusEnabled || status == dynamodb.TimeToLiveStatusEnabling {
			return nil
		}
	}

	_, err = r.client.UpdateTimeToLiveWithContext(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(r.tableName),
		TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
			AttributeName: aws.String("ttl"),
			Enabled:       aws.Bool(true),
		},
	})
	if err != nil {
		return fmt.Errorf("updating TTL: %w", err)
	}
	return nil
}
