package dynamodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.WebSocketRepository = (*WebSocketRepo)(nil)

type WebSocketRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewWebSocketRepo(client *dynamodb.DynamoDB, tableName string) *WebSocketRepo {
	return &WebSocketRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *WebSocketRepo) SaveConnection(ctx context.Context, conn *model.ConnectionInfo) error {
	if conn.TTL == 0 {
		conn.TTL = time.Now().Add(24 * time.Hour).Unix()
	}
	item, err := dynamodbattribute.MarshalMap(conn)
	if err != nil {
		return fmt.Errorf("marshaling connection: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("saving connection: %w", err)
	}

	return nil
}

func (r *WebSocketRepo) GetConnection(ctx context.Context, connectionID string) (*model.ConnectionInfo, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {S: aws.String(connectionID)},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}
	if result.Item == nil {
		return nil, model.ErrWebSocketConnectionNotFound
	}

	var conn model.ConnectionInfo
	if err := dynamodbattribute.UnmarshalMap(result.Item, &conn); err != nil {
		return nil, fmt.Errorf("unmarshaling connection: %w", err)
	}

	return &conn, nil
}

func (r *WebSocketRepo) DeleteConnection(ctx context.Context, connectionID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {S: aws.String(connectionID)},
		},
	}

	if _, err := r.client.DeleteItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("deleting connection: %w", err)
	}

	return nil
}

func (r *WebSocketRepo) UpdateSpot(ctx context.Context, connectionID, spot string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {S: aws.String(connectionID)},
		},
		UpdateExpression: aws.String("SET CurrentSpot = :spot, LastActive = :time, ttl = :ttl"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot": {S: aws.String(spot)},
			":time": {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
			":ttl":  {N: aws.String(fmt.Sprintf("%d", time.Now().Add(24 * time.Hour).Unix()))},
		},
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("updating connection spot: %w", err)
	}

	return nil
}

func (r *WebSocketRepo) UpdateLastActive(ctx context.Context, connectionID string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {S: aws.String(connectionID)},
		},
		UpdateExpression: aws.String("SET LastActive = :time, ttl = :ttl"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":time": {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
			":ttl":  {N: aws.String(fmt.Sprintf("%d", time.Now().Add(24 * time.Hour).Unix()))},
		},
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("updating connection last active: %w", err)
	}

	return nil
}

func (r *WebSocketRepo) GetConnectionsByUserIDs(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error) {
	if len(userIDs) == 0 {
		return []*model.ConnectionInfo{}, nil
	}

	exprParts := make([]string, 0, len(userIDs))
	exprValues := make(map[string]*dynamodb.AttributeValue, len(userIDs))
	for i, userID := range userIDs {
		key := fmt.Sprintf(":uid%d", i)
		exprParts = append(exprParts, key)
		exprValues[key] = &dynamodb.AttributeValue{S: aws.String(userID)}
	}

	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String(fmt.Sprintf("user_id IN (%s)", strings.Join(exprParts, ", "))),
		ExpressionAttributeValues: exprValues,
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning connections: %w", err)
	}

	connections := make([]*model.ConnectionInfo, 0, len(result.Items))
	for _, item := range result.Items {
		var conn model.ConnectionInfo
		if err := dynamodbattribute.UnmarshalMap(item, &conn); err != nil {
			return nil, fmt.Errorf("unmarshaling connection: %w", err)
		}
		connections = append(connections, &conn)
	}

	return connections, nil
}
