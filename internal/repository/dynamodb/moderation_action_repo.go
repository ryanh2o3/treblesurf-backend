package dynamodb

import (
	"context"
	"fmt"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

var _ repository.ModerationActionRepository = (*ModerationActionRepo)(nil)

// ModerationActionRepo implements ModerationActionRepository using DynamoDB.
type ModerationActionRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

// NewModerationActionRepo creates a new ModerationActionRepo.
func NewModerationActionRepo(client *dynamodb.DynamoDB, tableName string) *ModerationActionRepo {
	return &ModerationActionRepo{
		client:    client,
		tableName: tableName,
	}
}

// Create stores a new moderation action.
func (r *ModerationActionRepo) Create(ctx context.Context, action *model.ModerationAction) error {
	item, err := dynamodbattribute.MarshalMap(moderationActionItemFromModel(action))
	if err != nil {
		return fmt.Errorf("marshaling moderation action: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("creating moderation action: %w", err)
	}

	return nil
}

// GetByReportID retrieves all actions for a specific report.
func (r *ModerationActionRepo) GetByReportID(
	ctx context.Context,
	reportID string,
) ([]*model.ModerationAction, error) {
	keyCondition := expression.Key("report_id").Equal(expression.Value(reportID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, fmt.Errorf("building expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String("report_id-index"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying actions by report: %w", err)
	}

	return r.unmarshalActions(result.Items)
}

// List retrieves moderation actions with pagination.
func (r *ModerationActionRepo) List(
	ctx context.Context,
	limit, offset int,
) ([]*model.ModerationAction, error) {
	queryLimit := int64(limit + offset)
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		Limit:     aws.Int64(queryLimit),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning moderation actions: %w", err)
	}

	actions, err := r.unmarshalActions(result.Items)
	if err != nil {
		return nil, err
	}

	// Apply offset
	if offset >= len(actions) {
		return []*model.ModerationAction{}, nil
	}
	return actions[offset:], nil
}

func (r *ModerationActionRepo) unmarshalActions(
	items []map[string]*dynamodb.AttributeValue,
) ([]*model.ModerationAction, error) {
	actions := make([]*model.ModerationAction, 0, len(items))
	for _, item := range items {
		var actionItem moderationActionItem
		if err := dynamodbattribute.UnmarshalMap(item, &actionItem); err != nil {
			return nil, fmt.Errorf("unmarshaling action: %w", err)
		}
		actions = append(actions, actionItem.toModel())
	}
	return actions, nil
}
