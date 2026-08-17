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
	record := reportItemFromModel(report)
	if record.CountryRegionSpot == "" && report.Country != "" && report.Region != "" && report.Spot != "" {
		record.CountryRegionSpot = fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	}
	if record.DateReported == "" && !report.Timestamp.IsZero() {
		record.DateReported = report.Timestamp.UTC().Format(time.RFC3339)
	}
	if record.Time == "" && !report.Timestamp.IsZero() {
		record.Time = report.Timestamp.UTC().Format(time.RFC3339)
	}
	item, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
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
		var reportRecord reportItem
		if err := dynamodbattribute.UnmarshalMap(item, &reportRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, reportRecord.toModel())
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
		var reportRecord reportItem
		if err := dynamodbattribute.UnmarshalMap(item, &reportRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, reportRecord.toModel())
	}

	return reports, nil
}

func (r *ReportRepo) ScanSince(ctx context.Context, since time.Time, limit int) ([]*model.SurfReport, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("#Time > :cutoff"),
		ExpressionAttributeNames: map[string]*string{
			"#Time": aws.String("Time"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cutoff": {
				S: aws.String(since.UTC().Format("2006-01-02T15:04:05Z")),
			},
		},
	}

	if limit > 0 {
		input.Limit = aws.Int64(int64(limit))
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning reports: %w", err)
	}

	reports := make([]*model.SurfReport, 0, len(result.Items))
	for _, item := range result.Items {
		var reportRecord reportItem
		if err := dynamodbattribute.UnmarshalMap(item, &reportRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling report: %w", err)
		}
		reports = append(reports, reportRecord.toModel())
	}

	return reports, nil
}

func (r *ReportRepo) AnonymizeByUserEmail(ctx context.Context, email string) error {
	if email == "" {
		return nil
	}

	var startKey map[string]*dynamodb.AttributeValue
	for {
		input := &dynamodb.ScanInput{
			TableName:        aws.String(r.tableName),
			FilterExpression: aws.String("UserEmail = :email OR user_email = :email"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":email": {S: aws.String(email)},
			},
			ExclusiveStartKey: startKey,
		}

		result, err := r.client.ScanWithContext(ctx, input)
		if err != nil {
			return fmt.Errorf("scanning reports to anonymize: %w", err)
		}

		for _, item := range result.Items {
			if err := r.anonymizeReportItem(ctx, item); err != nil {
				return err
			}
		}

		if result.LastEvaluatedKey == nil {
			return nil
		}
		startKey = result.LastEvaluatedKey
	}
}

func (r *ReportRepo) anonymizeReportItem(ctx context.Context, item map[string]*dynamodb.AttributeValue) error {
	pk, ok := item["country_region_spot"]
	if !ok || pk == nil || pk.S == nil {
		return nil
	}
	sk, ok := item["dateReported"]
	if !ok || sk == nil || sk.S == nil {
		return nil
	}

	_, err := r.client.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"country_region_spot": pk,
			"dateReported":        sk,
		},
		UpdateExpression: aws.String("REMOVE UserEmail, user_email SET Reporter = :anon, reportedBy = :deleted"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":anon":    {S: aws.String("Anonymous")},
			":deleted": {S: aws.String("deleted")},
		},
	})
	if err != nil {
		return fmt.Errorf("anonymizing surf report: %w", err)
	}
	return nil
}
