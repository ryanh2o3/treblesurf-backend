package dynamodb

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
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
	item, err := dynamodbattribute.MarshalMap(session)
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
		return nil, model.ErrSessionNotFound
	}

	var session model.Session
	if err := dynamodbattribute.UnmarshalMap(result.Item, &session); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, model.ErrSessionExpired
	}

	return &session, nil
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
		var session model.Session
		if err := dynamodbattribute.UnmarshalMap(item, &session); err != nil {
			return nil, fmt.Errorf("unmarshaling session: %w", err)
		}
		if session.ExpiresAt.IsZero() || time.Now().Before(session.ExpiresAt) {
			sessions = append(sessions, &session)
		}
	}

	return sessions, nil
}
