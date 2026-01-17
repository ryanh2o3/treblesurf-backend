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

var _ repository.BuoyRepository = (*BuoyRepo)(nil)

type BuoyRepo struct {
	client             *dynamodb.DynamoDB
	dataTableName      string
	locationTableName  string
	regionPrefix       string
}

func NewBuoyRepo(client *dynamodb.DynamoDB, dataTableName, locationTableName string) *BuoyRepo {
	return &BuoyRepo{
		client:            client,
		dataTableName:     dataTableName,
		locationTableName: locationTableName,
		regionPrefix:      "Ireland",
	}
}

func (r *BuoyRepo) GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error) {
	regionBuoy := r.regionPrefix + "_" + buoyName
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.dataTableName),
		KeyConditionExpression: aws.String("region_buoy = :rb"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rb": {S: aws.String(regionBuoy)},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int64(1),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying live buoy data: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, model.ErrBuoyDataNotFound
	}

	var data model.BuoyData
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &data); err != nil {
		return nil, fmt.Errorf("unmarshaling buoy data: %w", err)
	}

	return &data, nil
}

func (r *BuoyRepo) GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error) {
	start := t.Add(-6 * time.Hour)
	end := t.Add(6 * time.Hour)

	data, err := r.GetDataRange(ctx, buoyName, start, end)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, model.ErrBuoyDataNotFound
	}

	return data[0], nil
}

func (r *BuoyRepo) GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error) {
	regionBuoy := r.regionPrefix + "_" + buoyName
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.dataTableName),
		KeyConditionExpression: aws.String("region_buoy = :rb AND dataDateTime BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":rb":    {S: aws.String(regionBuoy)},
			":start": {S: aws.String(start.UTC().Format(time.RFC3339))},
			":end":   {S: aws.String(end.UTC().Format(time.RFC3339))},
		},
		ScanIndexForward: aws.Bool(true),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying buoy data range: %w", err)
	}

	data := make([]*model.BuoyData, 0, len(result.Items))
	for _, item := range result.Items {
		var entry model.BuoyData
		if err := dynamodbattribute.UnmarshalMap(item, &entry); err != nil {
			return nil, fmt.Errorf("unmarshaling buoy data: %w", err)
		}
		data = append(data, &entry)
	}

	return data, nil
}

func (r *BuoyRepo) GetBatchDataRanges(ctx context.Context, requests []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
	// Group requests by buoy name and merge overlapping time ranges
	buoyRanges := make(map[string]struct{ start, end time.Time })
	for _, req := range requests {
		existing, ok := buoyRanges[req.BuoyName]
		if !ok {
			buoyRanges[req.BuoyName] = struct{ start, end time.Time }{req.Start, req.End}
			continue
		}
		// Merge ranges by taking min start and max end
		if req.Start.Before(existing.start) {
			existing.start = req.Start
		}
		if req.End.After(existing.end) {
			existing.end = req.End
		}
		buoyRanges[req.BuoyName] = existing
	}

	// Fetch data for each buoy with merged ranges
	results := make(map[string][]*model.BuoyData)
	for buoyName, timeRange := range buoyRanges {
		data, err := r.GetDataRange(ctx, buoyName, timeRange.start, timeRange.end)
		if err != nil {
			// Log but continue - some buoys may not have data
			continue
		}
		results[buoyName] = data
	}

	return results, nil
}

func (r *BuoyRepo) GetLocations(ctx context.Context) (map[string]*model.BuoyLocation, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.locationTableName),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning buoy locations: %w", err)
	}

	locations := make(map[string]*model.BuoyLocation)
	for _, item := range result.Items {
		var location model.BuoyLocation
		if err := dynamodbattribute.UnmarshalMap(item, &location); err != nil {
			return nil, fmt.Errorf("unmarshaling buoy location: %w", err)
		}
		key := location.Name
		if key == "" {
			if regionBuoyAttr, ok := item["region_buoy"]; ok && regionBuoyAttr.S != nil {
				parts := strings.Split(*regionBuoyAttr.S, "_")
				key = parts[len(parts)-1]
			}
		}
		if key != "" {
			locations[key] = &location
		}
	}

	return locations, nil
}
