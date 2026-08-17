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
		return nil, repository.ErrNotFound
	}

	dataRecord, err := parseBuoyData(result.Items[0])
	if err != nil {
		return nil, err
	}
	if dataRecord == nil {
		return nil, repository.ErrNotFound
	}

	return dataRecord, nil
}

func (r *BuoyRepo) GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error) {
	start := t.Add(-6 * time.Hour)
	end := t.Add(6 * time.Hour)

	data, err := r.GetDataRange(ctx, buoyName, start, end)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, repository.ErrNotFound
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
		entryRecord, err := parseBuoyData(item)
		if err != nil {
			return nil, err
		}
		if entryRecord != nil {
			data = append(data, entryRecord)
		}
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
		locationModel, key, err := parseBuoyLocation(item)
		if err != nil {
			return nil, err
		}
		if key != "" {
			locations[key] = locationModel
		}
	}

	return locations, nil
}

func parseBuoyData(item map[string]*dynamodb.AttributeValue) (*model.BuoyData, error) {
	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling buoy data: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	timestamp := parseBuoyTimestamp(raw)
	buoyName := firstString(raw, "name", "buoy_name", "BuoyName")
	if buoyName == "" {
		if regionBuoy := mapString(raw, "region_buoy", "RegionBuoy"); regionBuoy != "" {
			buoyName = buoyNameFromRegionBuoy(regionBuoy)
		}
	}

	// Python buoyData service (treblesurf-buoyData) writes PascalCase: MeanWaveDirection, SeaTemperature, AtmosphericPressure, etc.
	// Support both legacy snake_case and current writer attribute names.
	return &model.BuoyData{
		Timestamp:        timestamp,
		BuoyName:         buoyName,
		WaveHeight:       firstFloat(raw, "wave_height", "WaveHeight"),
		WavePeriod:       firstFloat(raw, "wave_period", "WavePeriod"),
		MaxPeriod:        firstFloat(raw, "max_period", "MaxPeriod"),
		WaveDirection:    firstFloat(raw, "wave_direction", "WaveDirection", "MeanWaveDirection"),
		WindSpeed:        firstFloat(raw, "wind_speed", "WindSpeed"),
		WindDirection:    firstFloat(raw, "wind_direction", "WindDirection"),
		Temperature:      firstFloat(raw, "temperature", "Temperature", "SeaTemperature"),
		Pressure:         firstFloat(raw, "pressure", "Pressure", "AtmosphericPressure"),
		SprTp:            firstFloat(raw, "SprTp"),
		ThTp:             firstFloat(raw, "ThTp"),
		MaxHeight:        firstFloat(raw, "max_height", "MaxHeight"),
		Gust:             firstFloat(raw, "Gust"),
		AirTemperature:   firstFloat(raw, "air_temperature", "AirTemperature"),
		DewPoint:         firstFloat(raw, "dew_point", "DewPoint"),
		RelativeHumidity: firstFloat(raw, "relative_humidity", "RelativeHumidity"),
		Salinity:         firstFloat(raw, "salinity", "Salinity"),
	}, nil
}

func parseBuoyTimestamp(raw map[string]interface{}) time.Time {
	timeValue := mapString(raw, "dataDateTime", "DataDateTime", "timestamp", "Timestamp")
	if timeValue == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, timeValue); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05Z", timeValue); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func parseBuoyLocation(item map[string]*dynamodb.AttributeValue) (*model.BuoyLocation, string, error) {
	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		return nil, "", fmt.Errorf("unmarshaling buoy location: %w", err)
	}

	regionBuoy := stringValue(raw["region_buoy"])
	name := firstString(raw, "name", "Name")
	if name == "" && regionBuoy != "" {
		name = buoyNameFromRegionBuoy(regionBuoy)
	}

	region := firstString(raw, "region", "Region")
	if region == "" && regionBuoy != "" {
		region = buoyRegionFromRegionBuoy(regionBuoy)
	}

	country := firstString(raw, "country", "Country")
	if country == "" && region != "" {
		country = region
	}

	spot := firstString(raw, "spot", "Spot")
	if spot == "" {
		spot = name
	}

	latitude := firstFloat(raw, "latitude", "Latitude")
	longitude := firstFloat(raw, "longitude", "Longitude")

	location := &model.BuoyLocation{
		Name:       name,
		RegionBuoy: regionBuoy,
		Country:    country,
		Region:     region,
		Spot:       spot,
		Latitude:   latitude,
		Longitude:  longitude,
	}

	key := name
	if key == "" {
		key = buoyNameFromRegionBuoy(regionBuoy)
	}
	return location, key, nil
}

func buoyNameFromRegionBuoy(regionBuoy string) string {
	if regionBuoy == "" {
		return ""
	}
	parts := strings.Split(regionBuoy, "_")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func buoyRegionFromRegionBuoy(regionBuoy string) string {
	if regionBuoy == "" {
		return ""
	}
	parts := strings.SplitN(regionBuoy, "_", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if str := stringValue(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func firstFloat(values map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if out, ok := floatValue(value); ok {
				return out
			}
		}
	}
	return 0
}
