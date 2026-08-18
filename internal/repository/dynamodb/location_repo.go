package dynamodb

import (
	"context"
	"fmt"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.LocationRepository = (*LocationRepo)(nil)

type LocationRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewLocationRepo(client *dynamodb.DynamoDB, tableName string) *LocationRepo {
	return &LocationRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *LocationRepo) GetRegions(ctx context.Context, country string) ([]string, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning locations: %w", err)
	}

	regions := make([]string, 0)
	seen := make(map[string]struct{})
	for _, item := range result.Items {
		location, err := parseLocationInfo(item)
		if err != nil {
			return nil, err
		}
		if location == nil {
			continue
		}
		parts := strings.Split(location.CountryRegionSpot, "/")
		if len(parts) < 2 || parts[0] != country {
			continue
		}
		region := parts[1]
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}

	return regions, nil
}

func (r *LocationRepo) GetSpots(ctx context.Context, country, region string) ([]*model.LocationInfo, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("begins_with(country_region_spot, :location)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {S: aws.String(fmt.Sprintf("%s/%s/", country, region))},
		},
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning spots: %w", err)
	}

	spots := make([]*model.LocationInfo, 0, len(result.Items))
	for _, item := range result.Items {
		location, err := parseLocationInfo(item)
		if err != nil {
			return nil, err
		}
		if location != nil {
			spots = append(spots, location)
		}
	}

	return spots, nil
}

func (r *LocationRepo) GetLocationInfo(ctx context.Context, country, region, spot string) (*model.LocationInfo, error) {
	locationKey := fmt.Sprintf("%s/%s/%s", country, region, spot)
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("country_region_spot = :location"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {S: aws.String(locationKey)},
		},
		Limit: aws.Int64(1),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying location: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, repository.ErrNotFound
	}

	location, err := parseLocationInfo(result.Items[0])
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, fmt.Errorf("location data missing")
	}

	return location, nil
}

func (r *LocationRepo) GetCoordinates(ctx context.Context, country, region, spot string) (lat, lon float64, err error) {
	location, err := r.GetLocationInfo(ctx, country, region, spot)
	if err != nil {
		return 0, 0, err
	}

	return location.Latitude, location.Longitude, nil
}

func parseLocationInfo(item map[string]*dynamodb.AttributeValue) (*model.LocationInfo, error) {
	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling location: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	location := &model.LocationInfo{
		CountryRegionSpot:   mapString(raw, "country_region_spot", "CountryRegionSpot"),
		IdealSwellDirection: mapString(raw, "IdealSwellDirection", "ideal_swell_direction"),
		Image:               mapString(raw, "Image", "image", "ImagePath", "image_path"),
		Type:                mapString(raw, "Type", "type"),
		ImageString:         mapString(raw, "ImageString", "image_string"),
		BeachDirection:      mapInt(raw, "BeachDirection", "beach_direction"),
		Elevation:           mapInt(raw, "Elevation", "elevation"),
		Latitude:            firstFloat(raw, "Latitude", "latitude"),
		Longitude:           firstFloat(raw, "Longitude", "longitude"),
	}

	return location, nil
}
