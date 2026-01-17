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

var _ repository.ReportRepository = (*ReportRepo)(nil)

type ReportRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewReportRepo(client *dynamodb.DynamoDB, tableName string) *ReportRepo {
	return &ReportRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *ReportRepo) Create(ctx context.Context, report *model.SurfReport) error {
	item, err := dynamodbattribute.MarshalMap(report)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	if report.Country != "" && report.Region != "" && report.Spot != "" {
		item["country_region_spot"] = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)),
		}
	}
	if !report.Timestamp.IsZero() {
		item["dateReported"] = &dynamodb.AttributeValue{
			S: aws.String(report.Timestamp.UTC().Format(time.RFC3339)),
		}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
		return fmt.Errorf("creating report: %w", err)
	}

	return nil
}

func (r *ReportRepo) GetBySpot(
	ctx context.Context,
	country, region, spot string,
	limit int,
) ([]*model.SurfReport, error) {
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", country, region, spot)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("country_region_spot = :crs"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crs": {S: aws.String(countryRegionSpot)},
		},
		ScanIndexForward: aws.Bool(false),
	}

	if limit > 0 {
		input.Limit = aws.Int64(int64(limit))
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying reports: %w", err)
	}

	reports := make([]*model.SurfReport, 0, len(result.Items))
	for _, item := range result.Items {
		var report model.SurfReport
		if err := dynamodbattribute.UnmarshalMap(item, &report); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, &report)
	}

	return reports, nil
}

func (r *ReportRepo) GetBySpotAndTimeRange(
	ctx context.Context,
	country, region, spot string,
	start, end time.Time,
) ([]*model.SurfReport, error) {
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", country, region, spot)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("country_region_spot = :crs"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crs": {S: aws.String(countryRegionSpot)},
			":start": {
				S: aws.String(start.UTC().Format(time.RFC3339)),
			},
			":end": {
				S: aws.String(end.UTC().Format(time.RFC3339)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#ts": aws.String("timestamp"),
		},
		FilterExpression: aws.String("#ts BETWEEN :start AND :end"),
		ScanIndexForward: aws.Bool(false),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying reports by time: %w", err)
	}

	reports := make([]*model.SurfReport, 0, len(result.Items))
	for _, item := range result.Items {
		var report model.SurfReport
		if err := dynamodbattribute.UnmarshalMap(item, &report); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, &report)
	}

	return reports, nil
}
