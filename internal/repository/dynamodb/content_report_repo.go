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
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

var _ repository.ContentReportRepository = (*ContentReportRepo)(nil)

// ContentReportRepo implements ContentReportRepository using DynamoDB.
type ContentReportRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

// NewContentReportRepo creates a new ContentReportRepo.
func NewContentReportRepo(client *dynamodb.DynamoDB, tableName string) *ContentReportRepo {
	return &ContentReportRepo{
		client:    client,
		tableName: tableName,
	}
}

// Create stores a new content report.
func (r *ContentReportRepo) Create(ctx context.Context, report *model.ContentReport) error {
	item, err := dynamodbattribute.MarshalMap(contentReportItemFromModel(report))
	if err != nil {
		return fmt.Errorf("marshaling content report: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("creating content report: %w", err)
	}

	return nil
}

// GetByID retrieves a content report by its ID.
func (r *ContentReportRepo) GetByID(ctx context.Context, id string) (*model.ContentReport, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting content report: %w", err)
	}
	if result.Item == nil {
		return nil, repository.ErrNotFound
	}

	var item contentReportItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshaling content report: %w", err)
	}

	return item.toModel(), nil
}

// GetBySurfReportID retrieves all reports for a specific surf report.
func (r *ContentReportRepo) GetBySurfReportID(
	ctx context.Context,
	surfReportID string,
) ([]*model.ContentReport, error) {
	keyCondition := expression.Key("surf_report_id").Equal(expression.Value(surfReportID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, fmt.Errorf("building expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String("surf_report_id-index"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying by surf report: %w", err)
	}

	return r.unmarshalReports(result.Items)
}

// GetByReporterID retrieves all reports submitted by a specific user.
func (r *ContentReportRepo) GetByReporterID(
	ctx context.Context,
	userID string,
) ([]*model.ContentReport, error) {
	keyCondition := expression.Key("reporter_user_id").Equal(expression.Value(userID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, fmt.Errorf("building expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String("reporter_user_id-index"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying by reporter: %w", err)
	}

	return r.unmarshalReports(result.Items)
}

// GetPendingReports retrieves pending reports for the moderation queue.
func (r *ContentReportRepo) GetPendingReports(
	ctx context.Context,
	limit, offset int,
) ([]*model.ContentReport, error) {
	keyCondition := expression.Key("status").Equal(expression.Value(string(model.ReportStatusPending)))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, fmt.Errorf("building expression: %w", err)
	}

	queryLimit := int64(limit + offset)
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String("status-created_at-index"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     aws.Int64(queryLimit),
		ScanIndexForward:          aws.Bool(true), // oldest first
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying pending reports: %w", err)
	}

	reports, err := r.unmarshalReports(result.Items)
	if err != nil {
		return nil, err
	}

	// Apply offset
	if offset >= len(reports) {
		return []*model.ContentReport{}, nil
	}
	return reports[offset:], nil
}

// UpdateStatus updates the status of a report.
func (r *ContentReportRepo) UpdateStatus(
	ctx context.Context,
	id, status, reviewedBy string,
) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		UpdateExpression: aws.String(
			"SET #status = :status, reviewed_by = :reviewedBy, updated_at = :updatedAt",
		),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status":     {S: aws.String(status)},
			":reviewedBy": {S: aws.String(reviewedBy)},
			":updatedAt":  {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		if isConditionalCheckFailed(err) {
			return repository.ErrNotFound
		}
		return fmt.Errorf("updating report status: %w", err)
	}

	return nil
}

// Resolve marks a report as resolved with the given resolution.
func (r *ContentReportRepo) Resolve(
	ctx context.Context,
	id, resolution, notes, reviewedBy string,
) error {
	now := time.Now().UTC()
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		UpdateExpression: aws.String(
			"SET #status = :status, resolution = :resolution, resolution_notes = :notes, " +
				"reviewed_by = :reviewedBy, reviewed_at = :reviewedAt, updated_at = :updatedAt",
		),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status":     {S: aws.String(string(model.ReportStatusResolved))},
			":resolution": {S: aws.String(resolution)},
			":notes":      {S: aws.String(notes)},
			":reviewedBy": {S: aws.String(reviewedBy)},
			":reviewedAt": {S: aws.String(now.Format(time.RFC3339))},
			":updatedAt":  {S: aws.String(now.Format(time.RFC3339))},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	}

	if _, err := r.client.UpdateItemWithContext(ctx, input); err != nil {
		if isConditionalCheckFailed(err) {
			return repository.ErrNotFound
		}
		return fmt.Errorf("resolving report: %w", err)
	}

	return nil
}

// CountByReporterSince counts reports submitted by a user since a given time.
func (r *ContentReportRepo) CountByReporterSince(
	ctx context.Context,
	userID string,
	since time.Time,
) (int, error) {
	keyCondition := expression.Key("reporter_user_id").Equal(expression.Value(userID)).
		And(expression.Key("created_at").GreaterThanEqual(expression.Value(since.Format(time.RFC3339))))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return 0, fmt.Errorf("building expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.tableName),
		IndexName:                 aws.String("reporter_user_id-created_at-index"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Select:                    aws.String("COUNT"),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("counting reports: %w", err)
	}

	return int(*result.Count), nil
}

func (r *ContentReportRepo) unmarshalReports(
	items []map[string]*dynamodb.AttributeValue,
) ([]*model.ContentReport, error) {
	reports := make([]*model.ContentReport, 0, len(items))
	for _, item := range items {
		var reportItem contentReportItem
		if err := dynamodbattribute.UnmarshalMap(item, &reportItem); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, reportItem.toModel())
	}
	return reports, nil
}
