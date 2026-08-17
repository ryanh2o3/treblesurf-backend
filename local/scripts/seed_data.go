// Package main provides a script to seed the local database with test data.
package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"treblesurf-backend/local/config"
	"treblesurf-backend/local/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const (
	surfForecastsTableName = "surf_forecasts"
	seedForecastSource     = "imi_swan+weatherkit"
)

func main() {
	// Load config
	cfg, err := config.Load(true)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	if err := storage.InitLocal(cfg); err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	if err := seedRealForecastData(); err != nil {
		log.Fatalf("Failed to seed forecast data: %v", err)
	}

	if err := seedBuoyData(); err != nil {
		log.Fatalf("Failed to seed buoy data: %v", err)
	}

	if err := seedUserData(); err != nil {
		log.Fatalf("Failed to seed user data: %v", err)
	}

	log.Println("Data seeding completed successfully")
}

// Helper functions for safe type assertions

func getString(m map[string]interface{}, key string) (string, error) {
	val, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %s not found", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", key)
	}
	return str, nil
}

//nolint:unparam // key parameter is needed for error messages
func getInt(m map[string]interface{}, key string) (int, error) {
	val, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("key %s not found", key)
	}
	i, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("key %s is not an int", key)
	}
	return i, nil
}

func getFloat64(m map[string]interface{}, key string) (float64, error) {
	val, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("key %s not found", key)
	}
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("key %s is not a float64", key)
	}
	return f, nil
}

func seedRealForecastData() error {
	log.Println("Seeding realistic forecast data for Donegal locations...")

	baseTime := time.Now().Truncate(24 * time.Hour)
	locations := getLocationDefinitions()

	if err := seedLocationData(locations); err != nil {
		return err
	}

	return seedForecastData(baseTime)
}

// getLocationDefinitions returns the location definitions with metadata.
func getLocationDefinitions() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"Ireland/Donegal/Ballyhiernan": {
			"BeachDirection":      200,
			"Latitude":            55.1938,
			"Longitude":           -8.1533,
			"Type":                "BeachBreak",
			"IdealSwellDirection": "NW",
			"ImagePath":           "../images/spotImages/Ireland/Donegal/BallyhiernanLow.jpg",
		},
		"Ireland/Donegal/Ballymastocker": {
			"BeachDirection":      170,
			"Latitude":            55.1826,
			"Longitude":           -7.7118,
			"Type":                "BeachBreak",
			"IdealSwellDirection": "N",
			"ImagePath":           "../images/spotImages/Ireland/Donegal/BallymastockerLow.jpg",
		},
		"Ireland/Donegal/Marble Hill": {
			"BeachDirection":      170,
			"Latitude":            55.1826,
			"Longitude":           -7.7118,
			"Type":                "BeachBreak",
			"IdealSwellDirection": "N",
			"ImagePath":           "../images/spotImages/Ireland/Donegal/Marble HillLow.jpg",
		},
		"Ireland/Donegal/Rossnowlagh": {
			"BeachDirection":      200,
			"Latitude":            55.1938,
			"Longitude":           -8.1533,
			"Type":                "BeachBreak",
			"IdealSwellDirection": "W",
			"ImagePath":           "../images/spotImages/Ireland/Donegal/RossnowlaghLow.jpg",
		},
		"Ireland/Donegal/Tullan Strand": {
			"BeachDirection":      170,
			"Latitude":            55.1826,
			"Longitude":           -7.7118,
			"Type":                "BeachBreak",
			"IdealSwellDirection": "W",
			"ImagePath":           "../images/spotImages/Ireland/Donegal/Tullan StrandLow.jpg",
		},
	}
}

// seedLocationData seeds location data entries with base64 encoded images.
func seedLocationData(locations map[string]map[string]interface{}) error {
	for spotID, metadata := range locations {
		imagePath, err := getString(metadata, "ImagePath")
		if err != nil {
			return fmt.Errorf("failed to get ImagePath for %s: %w", spotID, err)
		}
		base64Image, err := encodeImageToBase64(imagePath)
		if err != nil {
			log.Printf("Warning: Failed to encode image for %s: %v", spotID, err)
			base64Image = ""
		}

		locationItem := map[string]interface{}{
			"country_region_spot": spotID,
			"BeachDirection":      metadata["BeachDirection"],
			"Latitude":            metadata["Latitude"],
			"Longitude":           metadata["Longitude"],
			"Type":                metadata["Type"],
			"IdealSwellDirection": metadata["IdealSwellDirection"],
			"ImageString":         base64Image,
		}

		item, err := dynamodbattribute.MarshalMap(locationItem)
		if err != nil {
			return fmt.Errorf("failed to marshal location: %w", err)
		}

		_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String("LocationData"),
			Item:      item,
		})

		if err != nil {
			return fmt.Errorf("failed to put location item: %w", err)
		}

		log.Printf("Successfully seeded location data for %s with image (size: %d bytes)",
			spotID, len(base64Image))
	}
	return nil
}

// seedForecastData seeds forecast data for all spots.
func seedForecastData(baseTime time.Time) error {
	surfData1 := getSurfData1()
	surfData2 := getSurfData2()

	spotData := map[string][]map[string]interface{}{
		"Ireland#Donegal#Ballyhiernan":   surfData1,
		"Ireland#Donegal#Ballymastocker": surfData2,
		"Ireland#Donegal#Tullan Strand":  surfData1,
		"Ireland#Donegal#Marble Hill":    surfData2,
		"Ireland#Donegal#Rossnowlagh":    surfData1,
	}

	for spotID, forecastSamples := range spotData {
		log.Printf("Seeding forecast data for %s...", spotID)
		if err := seedSpotForecasts(spotID, forecastSamples, baseTime); err != nil {
			return err
		}
	}

	return nil
}

// getSurfData1 returns the first set of surf forecast data.
//
//nolint:dupl,funlen // Data structure, duplication and length acceptable for clarity
func getSurfData1() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"swellPeriod":           15.81,
			"waterTemperature":      7.92,
			"surfMessiness":         "Messy",
			"pressure":              1001.35,
			"waveEnergy":            10076.88,
			"precipitation":         0.01,
			"relativeWindDirection": "Cross",
			"swellHeight":           4.91,
			"swellDirection":        269.77,
			"temperature":           7.13,
			"directionQuality":      0.25,
			"humidity":              78.45,
			"surfSize":              7.19,
			"windDirection":         228.99,
			"windSpeed":             11.85,
			"hourOffset":            0,
		},
		{
			"swellPeriod":           14.32,
			"waterTemperature":      8.05,
			"surfMessiness":         "Clean",
			"pressure":              1003.62,
			"waveEnergy":            8697.41,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           4.53,
			"swellDirection":        272.18,
			"temperature":           8.27,
			"directionQuality":      0.32,
			"humidity":              76.21,
			"surfSize":              6.73,
			"windDirection":         112.47,
			"windSpeed":             8.21,
			"hourOffset":            6,
		},
		{
			"swellPeriod":           13.78,
			"waterTemperature":      8.12,
			"surfMessiness":         "Clean",
			"pressure":              1004.89,
			"waveEnergy":            7543.16,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           4.17,
			"swellDirection":        275.32,
			"temperature":           9.35,
			"directionQuality":      0.35,
			"humidity":              74.58,
			"surfSize":              6.25,
			"windDirection":         108.63,
			"windSpeed":             6.74,
			"hourOffset":            12,
		},
		{
			"swellPeriod":           12.95,
			"waterTemperature":      8.18,
			"surfMessiness":         "Clean",
			"pressure":              1006.17,
			"waveEnergy":            6218.53,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           3.86,
			"swellDirection":        278.45,
			"temperature":           8.92,
			"directionQuality":      0.38,
			"humidity":              75.12,
			"surfSize":              5.78,
			"windDirection":         115.24,
			"windSpeed":             5.92,
			"hourOffset":            18,
		},
		{
			"swellPeriod":           11.87,
			"waterTemperature":      8.21,
			"surfMessiness":         "Clean",
			"pressure":              1007.53,
			"waveEnergy":            5126.79,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           3.52,
			"swellDirection":        280.91,
			"temperature":           7.83,
			"directionQuality":      0.41,
			"humidity":              77.34,
			"surfSize":              5.34,
			"windDirection":         119.87,
			"windSpeed":             6.47,
			"hourOffset":            24,
		},
		{
			"swellPeriod":           10.43,
			"waterTemperature":      8.25,
			"surfMessiness":         "Slightly Bumpy",
			"pressure":              1008.42,
			"waveEnergy":            4275.38,
			"precipitation":         0.03,
			"relativeWindDirection": "Cross-Offshore",
			"swellHeight":           3.21,
			"swellDirection":        282.65,
			"temperature":           7.45,
			"directionQuality":      0.39,
			"humidity":              79.87,
			"surfSize":              4.85,
			"windDirection":         128.42,
			"windSpeed":             8.13,
			"hourOffset":            30,
		},
		{
			"swellPeriod":           9.67,
			"waterTemperature":      8.28,
			"surfMessiness":         "Bumpy",
			"pressure":              1007.89,
			"waveEnergy":            3582.94,
			"precipitation":         0.12,
			"relativeWindDirection": "Cross",
			"swellHeight":           2.96,
			"swellDirection":        284.18,
			"temperature":           8.26,
			"directionQuality":      0.36,
			"humidity":              82.45,
			"surfSize":              4.42,
			"windDirection":         136.78,
			"windSpeed":             10.57,
			"hourOffset":            36,
		},
		{
			"swellPeriod":           9.12,
			"waterTemperature":      8.31,
			"surfMessiness":         "Messy",
			"pressure":              1006.35,
			"waveEnergy":            2978.51,
			"precipitation":         0.28,
			"relativeWindDirection": "Cross-Onshore",
			"swellHeight":           2.74,
			"swellDirection":        285.92,
			"temperature":           9.34,
			"directionQuality":      0.33,
			"humidity":              85.12,
			"surfSize":              4.12,
			"windDirection":         142.35,
			"windSpeed":             13.24,
			"hourOffset":            42,
		},
		{
			"swellPeriod":           8.75,
			"waterTemperature":      8.33,
			"surfMessiness":         "Messy",
			"pressure":              1004.78,
			"waveEnergy":            2456.37,
			"precipitation":         0.45,
			"relativeWindDirection": "Onshore",
			"swellHeight":           2.48,
			"swellDirection":        286.43,
			"temperature":           10.12,
			"directionQuality":      0.29,
			"humidity":              87.56,
			"surfSize":              3.85,
			"windDirection":         187.21,
			"windSpeed":             15.78,
			"hourOffset":            48,
		},
	}
}

// getSurfData2 returns the second set of surf forecast data.
//
//nolint:dupl,funlen // Data structure, duplication and length acceptable for clarity
func getSurfData2() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"swellPeriod":           12.48,
			"waterTemperature":      8.15,
			"surfMessiness":         "Clean",
			"pressure":              1002.53,
			"waveEnergy":            3567.42,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           2.85,
			"swellDirection":        358.32,
			"temperature":           7.35,
			"directionQuality":      0.78,
			"humidity":              75.23,
			"surfSize":              3.75,
			"windDirection":         185.78,
			"windSpeed":             7.42,
			"hourOffset":            0,
		},
		{
			"swellPeriod":           11.97,
			"waterTemperature":      8.21,
			"surfMessiness":         "Clean",
			"pressure":              1004.18,
			"waveEnergy":            3125.87,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           2.68,
			"swellDirection":        356.47,
			"temperature":           8.12,
			"directionQuality":      0.81,
			"humidity":              73.48,
			"surfSize":              3.43,
			"windDirection":         178.35,
			"windSpeed":             6.83,
			"hourOffset":            6,
		},
		{
			"swellPeriod":           11.35,
			"waterTemperature":      8.26,
			"surfMessiness":         "Clean",
			"pressure":              1005.67,
			"waveEnergy":            2821.53,
			"precipitation":         0.00,
			"relativeWindDirection": "Offshore",
			"swellHeight":           2.47,
			"swellDirection":        354.21,
			"temperature":           9.24,
			"directionQuality":      0.83,
			"humidity":              71.92,
			"surfSize":              3.12,
			"windDirection":         175.42,
			"windSpeed":             5.76,
			"hourOffset":            12,
		},
		{
			"swellPeriod":           10.83,
			"waterTemperature":      8.29,
			"surfMessiness":         "Clean",
			"pressure":              1006.95,
			"waveEnergy":            2485.19,
			"precipitation":         0.02,
			"relativeWindDirection": "Cross-Offshore",
			"swellHeight":           2.32,
			"swellDirection":        351.86,
			"temperature":           8.67,
			"directionQuality":      0.79,
			"humidity":              72.37,
			"surfSize":              2.85,
			"windDirection":         165.23,
			"windSpeed":             6.34,
			"hourOffset":            18,
		},
		{
			"swellPeriod":           10.21,
			"waterTemperature":      8.31,
			"surfMessiness":         "Slightly Bumpy",
			"pressure":              1007.82,
			"waveEnergy":            2195.74,
			"precipitation":         0.05,
			"relativeWindDirection": "Cross",
			"swellHeight":           2.18,
			"swellDirection":        349.53,
			"temperature":           7.95,
			"directionQuality":      0.75,
			"humidity":              74.12,
			"surfSize":              2.64,
			"windDirection":         152.87,
			"windSpeed":             8.23,
			"hourOffset":            24,
		},
		{
			"swellPeriod":           9.65,
			"waterTemperature":      8.32,
			"surfMessiness":         "Bumpy",
			"pressure":              1008.25,
			"waveEnergy":            1925.38,
			"precipitation":         0.08,
			"relativeWindDirection": "Cross",
			"swellHeight":           2.03,
			"swellDirection":        347.24,
			"temperature":           7.56,
			"directionQuality":      0.71,
			"humidity":              76.89,
			"surfSize":              2.45,
			"windDirection":         145.32,
			"windSpeed":             9.87,
			"hourOffset":            30,
		},

		{
			"swellPeriod":           9.12,
			"waterTemperature":      8.33,
			"surfMessiness":         "Bumpy",
			"pressure":              1007.43,
			"waveEnergy":            1675.63,
			"precipitation":         0.15,
			"relativeWindDirection": "Cross-Onshore",
			"swellHeight":           1.87,
			"swellDirection":        345.18,
			"temperature":           8.42,
			"directionQuality":      0.67,
			"humidity":              79.34,
			"surfSize":              2.27,
			"windDirection":         138.75,
			"windSpeed":             11.42,
			"hourOffset":            36,
		},
		{
			"swellPeriod":           8.78,
			"waterTemperature":      8.34,
			"surfMessiness":         "Messy",
			"pressure":              1006.18,
			"waveEnergy":            1432.92,
			"precipitation":         0.23,
			"relativeWindDirection": "Onshore",
			"swellHeight":           1.75,
			"swellDirection":        343.45,
			"temperature":           9.18,
			"directionQuality":      0.63,
			"humidity":              82.57,
			"surfSize":              2.12,
			"windDirection":         95.34,
			"windSpeed":             13.78,
			"hourOffset":            42,
		},
		{
			"swellPeriod":           8.45,
			"waterTemperature":      8.35,
			"surfMessiness":         "Messy",
			"pressure":              1005.24,
			"waveEnergy":            1237.48,
			"precipitation":         0.32,
			"relativeWindDirection": "Onshore",
			"swellHeight":           1.62,
			"swellDirection":        341.92,
			"temperature":           9.87,
			"directionQuality":      0.59,
			"humidity":              85.12,
			"surfSize":              1.98,
			"windDirection":         88.67,
			"windSpeed":             15.34,
			"hourOffset":            48,
		},
	}
}

// seedSpotForecasts seeds forecast data for a single spot.
func seedSpotForecasts(spotID string, forecastSamples []map[string]interface{}, baseTime time.Time) error {
	for _, sample := range forecastSamples {
		if err := seedForecastSample(spotID, sample, baseTime); err != nil {
			return err
		}

		if !isLastSample(forecastSamples, sample) {
			nextSample, err := findNextSample(forecastSamples, sample)
			if err != nil {
				return fmt.Errorf("failed to find next sample: %w", err)
			}
			if err := seedInterpolatedForecasts(spotID, sample, nextSample, baseTime); err != nil {
				return err
			}
		}
	}
	return nil
}

// seedForecastSample seeds a single forecast sample.
func seedForecastSample(spotID string, sample map[string]interface{}, baseTime time.Time) error {
	hourOffset, err := getInt(sample, "hourOffset")
	if err != nil {
		return fmt.Errorf("failed to get hourOffset: %w", err)
	}
	forecastTime := baseTime.Add(time.Duration(hourOffset) * time.Hour)
	currentTime := time.Now()
	generatedAtTimestampStr := fmt.Sprintf("%d", currentTime.Unix())
	dateForecastedFor := forecastTime.Format("2006-01-02 15:04:05")
	timestampTS := forecastTime.Unix()

	dataMap, err := buildForecastDataMap(sample, dateForecastedFor)
	if err != nil {
		return fmt.Errorf("failed to build forecast data map: %w", err)
	}
	partitionKey := fmt.Sprintf("%s#%s#hourly", spotID, seedForecastSource)
	forecast := map[string]interface{}{
		"spot_id":      partitionKey,
		"timestamp_ts": timestampTS,
		"data":         dataMap,
		"generated_at": generatedAtTimestampStr,
		"source":       seedForecastSource,
	}

	item, err := dynamodbattribute.MarshalMap(forecast)
	if err != nil {
		return fmt.Errorf("failed to marshal forecast: %w", err)
	}

	_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(surfForecastsTableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put forecast item: %w", err)
	}
	return nil
}

// buildForecastDataMap builds the data map for a forecast sample.
//nolint:gocyclo,funlen // High complexity and length due to multiple type assertions needed
func buildForecastDataMap(sample map[string]interface{}, dateForecastedFor string) (map[string]interface{}, error) {
	swellPeriod, err := getFloat64(sample, "swellPeriod")
	if err != nil {
		return nil, err
	}
	waterTemperature, err := getFloat64(sample, "waterTemperature")
	if err != nil {
		return nil, err
	}
	surfMessiness, err := getString(sample, "surfMessiness")
	if err != nil {
		return nil, err
	}
	pressure, err := getFloat64(sample, "pressure")
	if err != nil {
		return nil, err
	}
	waveEnergy, err := getFloat64(sample, "waveEnergy")
	if err != nil {
		return nil, err
	}
	precipitation, err := getFloat64(sample, "precipitation")
	if err != nil {
		return nil, err
	}
	relativeWindDirection, err := getString(sample, "relativeWindDirection")
	if err != nil {
		return nil, err
	}
	swellHeight, err := getFloat64(sample, "swellHeight")
	if err != nil {
		return nil, err
	}
	swellDirection, err := getFloat64(sample, "swellDirection")
	if err != nil {
		return nil, err
	}
	temperature, err := getFloat64(sample, "temperature")
	if err != nil {
		return nil, err
	}
	directionQuality, err := getFloat64(sample, "directionQuality")
	if err != nil {
		return nil, err
	}
	humidity, err := getFloat64(sample, "humidity")
	if err != nil {
		return nil, err
	}
	surfSize, err := getFloat64(sample, "surfSize")
	if err != nil {
		return nil, err
	}
	windDirection, err := getFloat64(sample, "windDirection")
	if err != nil {
		return nil, err
	}
	windSpeed, err := getFloat64(sample, "windSpeed")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"swellPeriod":           swellPeriod,
		"waterTemperature":      waterTemperature,
		"surfMessiness":         surfMessiness,
		"pressure":              pressure,
		"waveEnergy":            waveEnergy,
		"precipitation":         precipitation,
		"relativeWindDirection": relativeWindDirection,
		"swellHeight":           swellHeight,
		"swellDirection":        swellDirection,
		"temperature":           temperature,
		"directionQuality":      directionQuality,
		"humidity":              humidity,
		"surfSize":              surfSize,
		"windDirection":         windDirection,
		"windSpeed":             windSpeed,
		"dateForecastedFor":     dateForecastedFor,
	}, nil
}

// seedInterpolatedForecasts seeds interpolated forecasts between two samples.
func seedInterpolatedForecasts(spotID string, sample, nextSample map[string]interface{}, baseTime time.Time) error {
	sampleOffset, err := getInt(sample, "hourOffset")
	if err != nil {
		return fmt.Errorf("failed to get sample hourOffset: %w", err)
	}
	nextOffset, err := getInt(nextSample, "hourOffset")
	if err != nil {
		return fmt.Errorf("failed to get nextSample hourOffset: %w", err)
	}
	hourDiff := nextOffset - sampleOffset
	if hourDiff <= 3 {
		return nil
	}

	steps := hourDiff/3 - 1
	currentTime := time.Now()
	generatedAtTimestampStr := fmt.Sprintf("%d", currentTime.Unix())

	for step := 1; step <= steps; step++ {
		progress := float64(step*3) / float64(hourDiff)
		stepHours := sampleOffset + step*3
		stepTime := baseTime.Add(time.Duration(stepHours) * time.Hour)
		stepTimestampTS := stepTime.Unix()
		stepDateForecastedFor := stepTime.Format("2006-01-02 15:04:05")

		interpDataMap, err := buildInterpolatedDataMap(sample, nextSample, progress, stepDateForecastedFor)
		if err != nil {
			return fmt.Errorf("failed to build interpolated data map: %w", err)
		}
		partitionKey := fmt.Sprintf("%s#%s#hourly", spotID, seedForecastSource)
		interpolatedForecast := map[string]interface{}{
			"spot_id":      partitionKey,
			"timestamp_ts": stepTimestampTS,
			"data":         interpDataMap,
			"generated_at": generatedAtTimestampStr,
			"source":       seedForecastSource,
		}

		item, err := dynamodbattribute.MarshalMap(interpolatedForecast)
		if err != nil {
			return fmt.Errorf("failed to marshal interpolated forecast: %w", err)
		}

		_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(surfForecastsTableName),
			Item:      item,
		})

		if err != nil {
			return fmt.Errorf("failed to put interpolated forecast item: %w", err)
		}
	}
	return nil
}

// buildInterpolatedDataMap builds an interpolated data map between two samples.
//
//nolint:gocyclo,funlen // High complexity and length due to multiple type assertions needed
func buildInterpolatedDataMap(
	sample, nextSample map[string]interface{},
	progress float64,
	dateForecastedFor string,
) (map[string]interface{}, error) {
	samplePeriod, err := getFloat64(sample, "swellPeriod")
	if err != nil {
		return nil, err
	}
	nextPeriod, err := getFloat64(nextSample, "swellPeriod")
	if err != nil {
		return nil, err
	}
	sampleWaterTemp, err := getFloat64(sample, "waterTemperature")
	if err != nil {
		return nil, err
	}
	nextWaterTemp, err := getFloat64(nextSample, "waterTemperature")
	if err != nil {
		return nil, err
	}
	samplePressure, err := getFloat64(sample, "pressure")
	if err != nil {
		return nil, err
	}
	nextPressure, err := getFloat64(nextSample, "pressure")
	if err != nil {
		return nil, err
	}
	sampleWaveEnergy, err := getFloat64(sample, "waveEnergy")
	if err != nil {
		return nil, err
	}
	nextWaveEnergy, err := getFloat64(nextSample, "waveEnergy")
	if err != nil {
		return nil, err
	}
	samplePrecip, err := getFloat64(sample, "precipitation")
	if err != nil {
		return nil, err
	}
	nextPrecip, err := getFloat64(nextSample, "precipitation")
	if err != nil {
		return nil, err
	}
	sampleSwellHeight, err := getFloat64(sample, "swellHeight")
	if err != nil {
		return nil, err
	}
	nextSwellHeight, err := getFloat64(nextSample, "swellHeight")
	if err != nil {
		return nil, err
	}
	sampleSwellDir, err := getFloat64(sample, "swellDirection")
	if err != nil {
		return nil, err
	}
	nextSwellDir, err := getFloat64(nextSample, "swellDirection")
	if err != nil {
		return nil, err
	}
	sampleTemp, err := getFloat64(sample, "temperature")
	if err != nil {
		return nil, err
	}
	nextTemp, err := getFloat64(nextSample, "temperature")
	if err != nil {
		return nil, err
	}
	sampleDirQuality, err := getFloat64(sample, "directionQuality")
	if err != nil {
		return nil, err
	}
	nextDirQuality, err := getFloat64(nextSample, "directionQuality")
	if err != nil {
		return nil, err
	}
	sampleHumidity, err := getFloat64(sample, "humidity")
	if err != nil {
		return nil, err
	}
	nextHumidity, err := getFloat64(nextSample, "humidity")
	if err != nil {
		return nil, err
	}
	sampleSurfSize, err := getFloat64(sample, "surfSize")
	if err != nil {
		return nil, err
	}
	nextSurfSize, err := getFloat64(nextSample, "surfSize")
	if err != nil {
		return nil, err
	}
	sampleWindDir, err := getFloat64(sample, "windDirection")
	if err != nil {
		return nil, err
	}
	nextWindDir, err := getFloat64(nextSample, "windDirection")
	if err != nil {
		return nil, err
	}
	sampleWindSpeed, err := getFloat64(sample, "windSpeed")
	if err != nil {
		return nil, err
	}
	nextWindSpeed, err := getFloat64(nextSample, "windSpeed")
	if err != nil {
		return nil, err
	}
	surfMessiness, err := getString(sample, "surfMessiness")
	if err != nil {
		return nil, err
	}
	relativeWindDirection, err := getString(sample, "relativeWindDirection")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"swellPeriod":           interpolate(samplePeriod, nextPeriod, progress),
		"waterTemperature":      interpolate(sampleWaterTemp, nextWaterTemp, progress),
		"surfMessiness":         surfMessiness,
		"pressure":              interpolate(samplePressure, nextPressure, progress),
		"waveEnergy":            interpolate(sampleWaveEnergy, nextWaveEnergy, progress),
		"precipitation":         interpolate(samplePrecip, nextPrecip, progress),
		"relativeWindDirection": relativeWindDirection,
		"swellHeight":           interpolate(sampleSwellHeight, nextSwellHeight, progress),
		"swellDirection":        interpolateAngle(sampleSwellDir, nextSwellDir, progress),
		"temperature":           interpolate(sampleTemp, nextTemp, progress),
		"directionQuality":      interpolate(sampleDirQuality, nextDirQuality, progress),
		"humidity":              interpolate(sampleHumidity, nextHumidity, progress),
		"surfSize":              interpolate(sampleSurfSize, nextSurfSize, progress),
		"windDirection":         interpolateAngle(sampleWindDir, nextWindDir, progress),
		"windSpeed":             interpolate(sampleWindSpeed, nextWindSpeed, progress),
		"dateForecastedFor":     dateForecastedFor,
	}, nil
}

// Helper function for linear interpolation
func interpolate(start, end, progress float64) float64 {
	return start + (end-start)*progress
}

// Helper function for interpolating angles (handles wraparound)
func interpolateAngle(start, end, progress float64) float64 {
	diff := math.Mod(end-start+540, 360) - 180
	return math.Mod(start+diff*progress+360, 360)
}

func seedUserData() error {
	log.Println("Seeding user data...")

	users := []map[string]interface{}{
		{
			"email":      "testuser@example.com",
			"GivenName":  "Test",
			"FamilyName": "User",
			"Theme":      "light",
			"Role":       "user",
		},
		{
			"email":      "admin@example.com",
			"GivenName":  "Admin",
			"FamilyName": "User",
			"Theme":      "dark",
			"Role":       "admin",
		},
	}

	for _, user := range users {
		item, err := dynamodbattribute.MarshalMap(user)
		if err != nil {
			return fmt.Errorf("failed to marshal user: %w", err)
		}

		_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String("Users"),
			Item:      item,
		})

		if err != nil {
			return fmt.Errorf("failed to put user item: %w", err)
		}
	}

	return nil
}

func seedBuoyData() error {
	log.Println("Seeding buoy data...")

	// First seed the buoy location data
	if err := seedBuoyLocationData(); err != nil {
		return fmt.Errorf("failed to seed buoy location data: %w", err)
	}

	// Then seed the buoy measurements
	if err := seedBuoyMeasurements(); err != nil {
		return fmt.Errorf("failed to seed buoy measurements: %w", err)
	}

	return nil
}

func seedBuoyLocationData() error {
	buoys := getBuoyLocationDefinitions()

	for _, buoy := range buoys {
		if err := seedSingleBuoyLocation(buoy); err != nil {
			return err
		}
	}

	log.Printf("Successfully seeded %d buoy locations", len(buoys))
	return nil
}

// getBuoyLocationDefinitions returns the buoy location definitions.
func getBuoyLocationDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"region_buoy": "NorthAtlantic_M6",
			"Latitude":    52.986,
			"Longitude":   -15.866,
			"Name":        "M6",
		},
		{
			"region_buoy": "NorthAtlantic_M3",
			"Latitude":    51.217,
			"Longitude":   -10.55,
			"Name":        "M3",
		},
		{
			"region_buoy": "NorthAtlantic_M4",
			"Latitude":    54.99972222,
			"Longitude":   -9.998888889,
			"Name":        "M4",
		},
		{
			"region_buoy": "NorthAtlantic_Blackstones",
			"Latitude":    56.06194444,
			"Longitude":   -7.056666667,
			"Name":        "Blackstones",
		},
		{
			"region_buoy": "NorthAtlantic_M2",
			"Latitude":    53.48,
			"Longitude":   -5.425,
			"Name":        "M2",
		},
		{
			"region_buoy": "NorthAtlantic_M5",
			"Latitude":    51.69,
			"Longitude":   -6.704,
			"Name":        "M5",
		},
		{
			"region_buoy": "NorthAtlantic_WestHebrides",
			"Latitude":    57.29194444,
			"Longitude":   -7.913888889,
			"Name":        "West Hebrides",
		},
	}
}

// seedSingleBuoyLocation seeds a single buoy location.
func seedSingleBuoyLocation(buoy map[string]interface{}) error {
	item, err := dynamodbattribute.MarshalMap(buoy)
	if err != nil {
		return fmt.Errorf("failed to marshal buoy location: %w", err)
	}

	_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("BuoyLocations"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put buoy location item: %w", err)
	}
	return nil
}

func seedBuoyMeasurements() error {
	now := time.Now().UTC().Truncate(time.Hour)
	measurementsTemplate := getBuoyMeasurementsTemplate()
	buoys := getBuoyDefinitions()

	totalCount := 0

	for _, buoy := range buoys {
		buoyMeasurements, err := generateBuoyMeasurements(buoy, measurementsTemplate, now)
		if err != nil {
			return err
		}

		if storeErr := storeBuoyMeasurements(buoyMeasurements); storeErr != nil {
			return storeErr
		}

		firstPeriod, err := getFloat64(buoyMeasurements[0], "WavePeriod")
		if err != nil {
			return fmt.Errorf("failed to get first WavePeriod: %w", err)
		}
		lastPeriod, err := getFloat64(buoyMeasurements[len(buoyMeasurements)-1], "WavePeriod")
		if err != nil {
			return fmt.Errorf("failed to get last WavePeriod: %w", err)
		}
		firstHeight, err := getFloat64(buoyMeasurements[0], "WaveHeight")
		if err != nil {
			return fmt.Errorf("failed to get first WaveHeight: %w", err)
		}
		lastHeight, err := getFloat64(buoyMeasurements[len(buoyMeasurements)-1], "WaveHeight")
		if err != nil {
			return fmt.Errorf("failed to get last WaveHeight: %w", err)
		}
		log.Printf(
			"Successfully seeded %d measurements for buoy %s (period range: %.1f-%.1fs, height range: %.1f-%.1fm)",
			len(buoyMeasurements),
			buoy["name"],
			firstPeriod,
			lastPeriod,
			firstHeight,
			lastHeight)

		totalCount += len(buoyMeasurements)
	}

	log.Printf("Successfully seeded %d total buoy measurements", totalCount)
	return nil
}

// getBuoyMeasurementsTemplate returns the template for buoy measurements.
//
//nolint:funlen // Large data structure, cannot be meaningfully split
func getBuoyMeasurementsTemplate() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      9.531,
			"AtmosphericPressure": 995.496,
			"DewPoint":            nil,
			"Gust":                37.228,
			"MaxHeight":           14.219,
			"MaxPeriod":           18.164,
			"MeanWaveDirection":   262,
			"name":                "M4",
			"RelativeHumidity":    71.777,
			"Salinity":            35.9893,
			"SeaTemperature":      10.979,
			"SprTp":               98.438,
			"ThTp":                274.219,
			"WaveHeight":          8.32,
			"WavePeriod":          10.547,
			"WindDirection":       243,
			"WindSpeed":           27.437,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      10.166,
			"AtmosphericPressure": 997.729,
			"DewPoint":            nil,
			"Gust":                35.976,
			"MaxHeight":           12.656,
			"MaxPeriod":           14.297,
			"MeanWaveDirection":   264,
			"name":                "M4",
			"RelativeHumidity":    67.09,
			"Salinity":            35.9906,
			"SeaTemperature":      10.992,
			"SprTp":               80.156,
			"ThTp":                268.594,
			"WaveHeight":          8.203,
			"WavePeriod":          10.43,
			"WindDirection":       242,
			"WindSpeed":           28.462,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      9.043,
			"AtmosphericPressure": 1002.112,
			"DewPoint":            nil,
			"Gust":                29.714,
			"MaxHeight":           10.625,
			"MaxPeriod":           14.297,
			"MeanWaveDirection":   257,
			"name":                "M4",
			"RelativeHumidity":    72.07,
			"Salinity":            35.99434,
			"SeaTemperature":      11.019,
			"SprTp":               99.844,
			"ThTp":                260.156,
			"WaveHeight":          5.977,
			"WavePeriod":          9.258,
			"WindDirection":       264,
			"WindSpeed":           22.2,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      8.506,
			"AtmosphericPressure": 1002.856,
			"DewPoint":            nil,
			"Gust":                24.705,
			"MaxHeight":           9.063,
			"MaxPeriod":           12.539,
			"MeanWaveDirection":   259,
			"name":                "M4",
			"RelativeHumidity":    74.512,
			"Salinity":            35.99617,
			"SeaTemperature":      11.032,
			"SprTp":               116.719,
			"ThTp":                255.938,
			"WaveHeight":          5.156,
			"WavePeriod":          8.438,
			"WindDirection":       281,
			"WindSpeed":           19.126,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      8.945,
			"AtmosphericPressure": 1003.125,
			"DewPoint":            nil,
			"Gust":                35.407,
			"MaxHeight":           9.375,
			"MaxPeriod":           15.352,
			"MeanWaveDirection":   257,
			"name":                "M4",
			"RelativeHumidity":    71.582,
			"Salinity":            35.99792,
			"SeaTemperature":      11.03,
			"SprTp":               168.75,
			"ThTp":                262.969,
			"WaveHeight":          5.156,
			"WavePeriod":          8.32,
			"WindDirection":       260,
			"WindSpeed":           23.794,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      8.896,
			"AtmosphericPressure": 1004.041,
			"DewPoint":            nil,
			"Gust":                31.08,
			"MaxHeight":           8.438,
			"MaxPeriod":           12.539,
			"MeanWaveDirection":   263,
			"name":                "M4",
			"RelativeHumidity":    75.293,
			"Salinity":            35.99442,
			"SeaTemperature":      11.027,
			"SprTp":               132.188,
			"ThTp":                260.156,
			"WaveHeight":          5.742,
			"WavePeriod":          8.906,
			"WindDirection":       271,
			"WindSpeed":           22.997,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      9.775,
			"AtmosphericPressure": 1008.569,
			"DewPoint":            nil,
			"Gust":                31.422,
			"MaxHeight":           5.469,
			"MaxPeriod":           9.141,
			"MeanWaveDirection":   345,
			"name":                "M4",
			"RelativeHumidity":    89.355,
			"Salinity":            35.92728,
			"SeaTemperature":      11.118,
			"SprTp":               118.125,
			"ThTp":                337.5,
			"WaveHeight":          3.164,
			"WavePeriod":          6.328,
			"WindDirection":       22,
			"WindSpeed":           24.136,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      9.922,
			"AtmosphericPressure": 1008.765,
			"DewPoint":            nil,
			"Gust":                33.585,
			"MaxHeight":           6.25,
			"MaxPeriod":           9.961,
			"MeanWaveDirection":   350,
			"name":                "M4",
			"RelativeHumidity":    83.105,
			"Salinity":            35.92506,
			"SeaTemperature":      11.113,
			"SprTp":               151.875,
			"ThTp":                338.906,
			"WaveHeight":          3.398,
			"WavePeriod":          6.563,
			"WindDirection":       24,
			"WindSpeed":           23.225,
		},
		{
			"region_buoy":         "Ireland_M4",
			"AirTemperature":      9.873,
			"AtmosphericPressure": 1008.752,
			"DewPoint":            nil,
			"Gust":                33.357,
			"MaxHeight":           5.313,
			"MaxPeriod":           9.961,
			"MeanWaveDirection":   352,
			"name":                "M4",
			"RelativeHumidity":    81.445,
			"Salinity":            35.92529,
			"SeaTemperature":      11.104,
			"SprTp":               109.688,
			"ThTp":                341.719,
			"WaveHeight":          3.633,
			"WavePeriod":          6.68,
			"WindDirection":       29,
			"WindSpeed":           23.908,
		},
	}
}

// getBuoyDefinitions returns the buoy definitions with variation factors.
func getBuoyDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"region_buoy":        "Ireland_M6",
			"name":               "M6",
			"wave_height_factor": 1.35,
			"wave_period_offset": 3.0,
			"max_height_factor":  1.4,
		},
		{
			"region_buoy": "Ireland_M3",
			"name":        "M3",
			// South-west of Ireland, also sees larger swells
			"wave_height_factor": 1.25,
			"wave_period_offset": 2.5,
			"max_height_factor":  1.3,
		},
		{
			"region_buoy": "Ireland_M4",
			"name":        "M4",
			// Northwest coast, our baseline
			"wave_height_factor": 1.0,
			"wave_period_offset": 0.0,
			"max_height_factor":  1.0,
		},
		{
			"region_buoy": "Ireland_Blackstones",
			"name":        "Blackstones",
			// Western approaches - sees large swells
			"wave_height_factor": 1.4,
			"wave_period_offset": 2.8,
			"max_height_factor":  1.45,
		},
		{
			"region_buoy": "Ireland_M2",
			"name":        "M2",
			// East coast, much more sheltered
			"wave_height_factor": 0.6,
			"wave_period_offset": -1.0,
			"max_height_factor":  0.65,
		},
		{
			"region_buoy": "Ireland_M5",
			"name":        "M5",
			// Southeast coast, somewhat sheltered
			"wave_height_factor": 0.75,
			"wave_period_offset": -0.5,
			"max_height_factor":  0.8,
		},
		{
			"region_buoy": "Ireland_WestHebrides",
			"name":        "West Hebrides",
			// Further north, exposed to Atlantic swells
			"wave_height_factor": 1.2,
			"wave_period_offset": 1.5,
			"max_height_factor":  1.25,
		},
	}
}

// generateBuoyMeasurements generates measurement data for a buoy.
//

func generateBuoyMeasurements(
	buoy map[string]interface{},
	measurementsTemplate []map[string]interface{},
	now time.Time,
) ([]map[string]interface{}, error) {
	buoyMeasurements := make([]map[string]interface{}, len(measurementsTemplate))
	waveHeightFactor, err := getFloat64(buoy, "wave_height_factor")
	if err != nil {
		return nil, fmt.Errorf("failed to get wave_height_factor: %w", err)
	}
	wavePeriodOffset, err := getFloat64(buoy, "wave_period_offset")
	if err != nil {
		return nil, fmt.Errorf("failed to get wave_period_offset: %w", err)
	}
	maxHeightFactor, err := getFloat64(buoy, "max_height_factor")
	if err != nil {
		return nil, fmt.Errorf("failed to get max_height_factor: %w", err)
	}

	for i, template := range measurementsTemplate {
		measurement := cloneMeasurementTemplate(template)
		measurement["region_buoy"] = buoy["region_buoy"]
		measurement["name"] = buoy["name"]

		if err := applyWaveVariations(measurement, waveHeightFactor, wavePeriodOffset, maxHeightFactor); err != nil {
			return nil, fmt.Errorf("failed to apply wave variations: %w", err)
		}

		hourOffset := len(measurementsTemplate) - 1 - i
		timestamp := now.Add(time.Duration(-hourOffset) * time.Hour)
		measurement["dataDateTime"] = timestamp.Format(time.RFC3339)

		buoyMeasurements[i] = measurement
	}

	return buoyMeasurements, nil
}

// cloneMeasurementTemplate clones a measurement template.
func cloneMeasurementTemplate(template map[string]interface{}) map[string]interface{} {
	measurement := make(map[string]interface{})
	for k, v := range template {
		measurement[k] = v
	}
	return measurement
}

// applyWaveVariations applies wave variations to a measurement.
func applyWaveVariations(
	measurement map[string]interface{},
	waveHeightFactor, wavePeriodOffset, maxHeightFactor float64,
) error {
	basePeriod, err := getFloat64(measurement, "WavePeriod")
	if err != nil {
		return err
	}
	measurement["WavePeriod"] = math.Max(8.0, math.Min(16.0, basePeriod+wavePeriodOffset))

	baseMaxPeriod, err := getFloat64(measurement, "MaxPeriod")
	if err != nil {
		return err
	}
	measurement["MaxPeriod"] = math.Max(10.0, math.Min(18.0, baseMaxPeriod+wavePeriodOffset))

	baseHeight, err := getFloat64(measurement, "WaveHeight")
	if err != nil {
		return err
	}
	measurement["WaveHeight"] = math.Max(0.2, math.Min(5.0, baseHeight*waveHeightFactor))

	baseMaxHeight, err := getFloat64(measurement, "MaxHeight")
	if err != nil {
		return err
	}
	measurement["MaxHeight"] = math.Max(0.5, math.Min(8.0, baseMaxHeight*maxHeightFactor))
	return nil
}

// storeBuoyMeasurements stores buoy measurements in DynamoDB.
func storeBuoyMeasurements(buoyMeasurements []map[string]interface{}) error {
	for _, measurement := range buoyMeasurements {
		item, err := dynamodbattribute.MarshalMap(measurement)
		if err != nil {
			return fmt.Errorf("failed to marshal buoy measurement: %w", err)
		}

		_, err = storage.DB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String("BuoyData"),
			Item:      item,
		})

		if err != nil {
			return fmt.Errorf("failed to put buoy measurement item: %w", err)
		}
	}
	return nil
}

// Helper function to check if a sample is the last one in the array
func isLastSample(samples []map[string]interface{}, current map[string]interface{}) bool {
	lastSample := samples[len(samples)-1]
	return current["hourOffset"] == lastSample["hourOffset"]
}

// Helper function to find the next sample after the current one
func findNextSample(samples []map[string]interface{}, current map[string]interface{}) (map[string]interface{}, error) {
	currentOffset, err := getInt(current, "hourOffset")
	if err != nil {
		return nil, fmt.Errorf("failed to get current hourOffset: %w", err)
	}
	var nextSample map[string]interface{}
	minOffsetDiff := math.MaxInt32

	// Find the sample with the smallest hour offset greater than the current sample
	for _, sample := range samples {
		sampleOffset, err := getInt(sample, "hourOffset")
		if err != nil {
			return nil, fmt.Errorf("failed to get sample hourOffset: %w", err)
		}
		if sampleOffset > currentOffset && sampleOffset-currentOffset < minOffsetDiff {
			minOffsetDiff = sampleOffset - currentOffset
			nextSample = sample
		}
	}

	// If no next sample found (should not happen in normal operation), return the first sample
	if nextSample == nil {
		return samples[0], nil
	}

	return nextSample, nil
}

// Helper function to encode an image file to a base64 string
func encodeImageToBase64(imagePath string) (string, error) {
	// Validate and clean the file path to prevent directory traversal
	cleanedPath := filepath.Clean(imagePath)
	// Ensure the path doesn't contain directory traversal sequences
	if strings.Contains(cleanedPath, "..") {
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Open the image file
	file, err := os.Open(cleanedPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close image file: %v", closeErr)
		}
	}()

	// Read the file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read image file content: %w", err)
	}

	// Encode the file content to base64
	base64Image := base64.StdEncoding.EncodeToString(fileContent)
	return base64Image, nil
}
