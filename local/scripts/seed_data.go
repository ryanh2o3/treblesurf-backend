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

func seedRealForecastData() error {
    log.Println("Seeding realistic forecast data for Donegal locations...")
    
    // Create timestamp for base forecast time
    baseTime := time.Now().Truncate(24 * time.Hour)
    
    // Define locations with their metadata and image paths
    locations := map[string]map[string]interface{}{
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
    
    // First, create location data entries with base64 encoded images
    for spotID, metadata := range locations {
        // Read and encode the image
        imagePath := metadata["ImagePath"].(string)
        base64Image, err := encodeImageToBase64(imagePath)
        if err != nil {
            log.Printf("Warning: Failed to encode image for %s: %v", spotID, err)
            base64Image = "" // Set empty string if image encoding fails
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
    
    surfData1 := []map[string]interface{}{
    {
        "swellPeriod":        15.81,
        "waterTemperature":   7.92,
        "surfMessiness":      "Messy",
        "pressure":           1001.35,
        "waveEnergy":         10076.88,
        "precipitation":      0.01,
        "relativeWindDirection": "Cross",
        "swellHeight":        4.91,
        "swellDirection":     269.77,
        "temperature":        7.13,
        "directionQuality":   0.25,
        "humidity":           78.45,
        "surfSize":           7.19,
        "windDirection":      228.99,
        "windSpeed":          11.85,
        "hourOffset":         0,
    },
    {
        "swellPeriod":        14.32,
        "waterTemperature":   8.05,
        "surfMessiness":      "Clean",
        "pressure":           1003.62,
        "waveEnergy":         8697.41,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        4.53,
        "swellDirection":     272.18,
        "temperature":        8.27,
        "directionQuality":   0.32,
        "humidity":           76.21,
        "surfSize":           6.73,
        "windDirection":      112.47,
        "windSpeed":          8.21,
        "hourOffset":         6,
    },
    {
        "swellPeriod":        13.78,
        "waterTemperature":   8.12,
        "surfMessiness":      "Clean",
        "pressure":           1004.89,
        "waveEnergy":         7543.16,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        4.17,
        "swellDirection":     275.32,
        "temperature":        9.35,
        "directionQuality":   0.35,
        "humidity":           74.58,
        "surfSize":           6.25,
        "windDirection":      108.63,
        "windSpeed":          6.74,
        "hourOffset":         12,
    },
    {
        "swellPeriod":        12.95,
        "waterTemperature":   8.18,
        "surfMessiness":      "Clean",
        "pressure":           1006.17,
        "waveEnergy":         6218.53,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        3.86,
        "swellDirection":     278.45,
        "temperature":        8.92,
        "directionQuality":   0.38,
        "humidity":           75.12,
        "surfSize":           5.78,
        "windDirection":      115.24,
        "windSpeed":          5.92,
        "hourOffset":         18,
    },
    {
        "swellPeriod":        11.87,
        "waterTemperature":   8.21,
        "surfMessiness":      "Clean",
        "pressure":           1007.53,
        "waveEnergy":         5126.79,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        3.52,
        "swellDirection":     280.91,
        "temperature":        7.83,
        "directionQuality":   0.41,
        "humidity":           77.34,
        "surfSize":           5.34,
        "windDirection":      119.87,
        "windSpeed":          6.47,
        "hourOffset":         24,
    },
    {
        "swellPeriod":        10.43,
        "waterTemperature":   8.25,
        "surfMessiness":      "Slightly Bumpy",
        "pressure":           1008.42,
        "waveEnergy":         4275.38,
        "precipitation":      0.03,
        "relativeWindDirection": "Cross-Offshore",
        "swellHeight":        3.21,
        "swellDirection":     282.65,
        "temperature":        7.45,
        "directionQuality":   0.39,
        "humidity":           79.87,
        "surfSize":           4.85,
        "windDirection":      128.42,
        "windSpeed":          8.13,
        "hourOffset":         30,
    },
    {
        "swellPeriod":        9.67,
        "waterTemperature":   8.28,
        "surfMessiness":      "Bumpy",
        "pressure":           1007.89,
        "waveEnergy":         3582.94,
        "precipitation":      0.12,
        "relativeWindDirection": "Cross",
        "swellHeight":        2.96,
        "swellDirection":     284.18,
        "temperature":        8.26,
        "directionQuality":   0.36,
        "humidity":           82.45,
        "surfSize":           4.42,
        "windDirection":      136.78,
        "windSpeed":          10.57,
        "hourOffset":         36,
    },
    {
        "swellPeriod":        9.12,
        "waterTemperature":   8.31,
        "surfMessiness":      "Messy",
        "pressure":           1006.35,
        "waveEnergy":         2978.51,
        "precipitation":      0.28,
        "relativeWindDirection": "Cross-Onshore",
        "swellHeight":        2.74,
        "swellDirection":     285.92,
        "temperature":        9.34,
        "directionQuality":   0.33,
        "humidity":           85.12,
        "surfSize":           4.12,
        "windDirection":      142.35,
        "windSpeed":          13.24,
        "hourOffset":         42,
    },
    {
        "swellPeriod":        8.75,
        "waterTemperature":   8.33,
        "surfMessiness":      "Messy",
        "pressure":           1004.78,
        "waveEnergy":         2456.37,
        "precipitation":      0.45,
        "relativeWindDirection": "Onshore",
        "swellHeight":        2.48,
        "swellDirection":     286.43,
        "temperature":        10.12,
        "directionQuality":   0.29,
        "humidity":           87.56,
        "surfSize":           3.85,
        "windDirection":      187.21,
        "windSpeed":          15.78,
        "hourOffset":         48,
    },
}

// Sample forecast data for Ballymastocker Bay
surfData2 := []map[string]interface{}{
    {
        "swellPeriod":        12.48,
        "waterTemperature":   8.15,
        "surfMessiness":      "Clean",
        "pressure":           1002.53,
        "waveEnergy":         3567.42,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        2.85,
        "swellDirection":     358.32,
        "temperature":        7.35,
        "directionQuality":   0.78,
        "humidity":           75.23,
        "surfSize":           3.75,
        "windDirection":      185.78,
        "windSpeed":          7.42,
        "hourOffset":         0,
    },
    {
        "swellPeriod":        11.97,
        "waterTemperature":   8.21,
        "surfMessiness":      "Clean",
        "pressure":           1004.18,
        "waveEnergy":         3125.87,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        2.68,
        "swellDirection":     356.47,
        "temperature":        8.12,
        "directionQuality":   0.81,
        "humidity":           73.48,
        "surfSize":           3.43,
        "windDirection":      178.35,
        "windSpeed":          6.83,
        "hourOffset":         6,
    },
    {
        "swellPeriod":        11.35,
        "waterTemperature":   8.26,
        "surfMessiness":      "Clean",
        "pressure":           1005.67,
        "waveEnergy":         2821.53,
        "precipitation":      0.00,
        "relativeWindDirection": "Offshore",
        "swellHeight":        2.47,
        "swellDirection":     354.21,
        "temperature":        9.24,
        "directionQuality":   0.83,
        "humidity":           71.92,
        "surfSize":           3.12,
        "windDirection":      175.42,
        "windSpeed":          5.76,
        "hourOffset":         12,
    },
    {
        "swellPeriod":        10.83,
        "waterTemperature":   8.29,
        "surfMessiness":      "Clean",
        "pressure":           1006.95,
        "waveEnergy":         2485.19,
        "precipitation":      0.02,
        "relativeWindDirection": "Cross-Offshore",
        "swellHeight":        2.32,
        "swellDirection":     351.86,
        "temperature":        8.67,
        "directionQuality":   0.79,
        "humidity":           72.37,
        "surfSize":           2.85,
        "windDirection":      165.23,
        "windSpeed":          6.34,
        "hourOffset":         18,
    },
    {
        "swellPeriod":        10.21,
        "waterTemperature":   8.31,
        "surfMessiness":      "Slightly Bumpy",
        "pressure":           1007.82,
        "waveEnergy":         2195.74,
        "precipitation":      0.05,
        "relativeWindDirection": "Cross",
        "swellHeight":        2.18,
        "swellDirection":     349.53,
        "temperature":        7.95,
        "directionQuality":   0.75,
        "humidity":           74.12,
        "surfSize":           2.64,
        "windDirection":      152.87,
        "windSpeed":          8.23,
        "hourOffset":         24,
    },
    {
        "swellPeriod":        9.65,
        "waterTemperature":   8.32,
        "surfMessiness":      "Bumpy",
        "pressure":           1008.25,
        "waveEnergy":         1925.38,
        "precipitation":      0.08,
        "relativeWindDirection": "Cross",
        "swellHeight":        2.03,
        "swellDirection":     347.24,
        "temperature":        7.56,
        "directionQuality":   0.71,
        "humidity":           76.89,
        "surfSize":           2.45,
        "windDirection":      145.32,
        "windSpeed":          9.87,
        "hourOffset":         30,
    },
    
    {
        "swellPeriod":        9.12,
        "waterTemperature":   8.33,
        "surfMessiness":      "Bumpy",
        "pressure":           1007.43,
        "waveEnergy":         1675.63,
        "precipitation":      0.15,
        "relativeWindDirection": "Cross-Onshore",
        "swellHeight":        1.87,
        "swellDirection":     345.18,
        "temperature":        8.42,
        "directionQuality":   0.67,
        "humidity":           79.34,
        "surfSize":           2.27,
        "windDirection":      138.75,
        "windSpeed":          11.42,
        "hourOffset":         36,
    },
    {
        "swellPeriod":        8.78,
        "waterTemperature":   8.34,
        "surfMessiness":      "Messy",
        "pressure":           1006.18,
        "waveEnergy":         1432.92,
        "precipitation":      0.23,
        "relativeWindDirection": "Onshore",
        "swellHeight":        1.75,
        "swellDirection":     343.45,
        "temperature":        9.18,
        "directionQuality":   0.63,
        "humidity":           82.57,
        "surfSize":           2.12,
        "windDirection":      95.34,
        "windSpeed":          13.78,
        "hourOffset":         42,
    },
    {
        "swellPeriod":        8.45,
        "waterTemperature":   8.35,
        "surfMessiness":      "Messy",
        "pressure":           1005.24,
        "waveEnergy":         1237.48,
        "precipitation":      0.32,
        "relativeWindDirection": "Onshore",
        "swellHeight":        1.62,
        "swellDirection":     341.92,
        "temperature":        9.87,
        "directionQuality":   0.59,
        "humidity":           85.12,
        "surfSize":           1.98,
        "windDirection":      88.67,
        "windSpeed":          15.34,
        "hourOffset":         48,
    },
}
    
    // Create the forecasts
    spotData := map[string][]map[string]interface{}{
        "Ireland#Donegal#Ballyhiernan": surfData1,
        "Ireland#Donegal#Ballymastocker": surfData2,
        "Ireland#Donegal#Tullan Strand": surfData1,
        "Ireland#Donegal#Marble Hill": surfData2,
        "Ireland#Donegal#Rossnowlagh": surfData1,
    }
    
    for spotID, forecastSamples := range spotData {
        log.Printf("Seeding forecast data for %s...", spotID)
        
        for _, sample := range forecastSamples {
            // Calculate the forecast time
            forecastTime := baseTime.Add(time.Duration(sample["hourOffset"].(int)) * time.Hour)
            currentTime := time.Now()
            generatedAtTimestampStr := fmt.Sprintf("%d", currentTime.Unix())
            // Get the nearest previous hour for forecast_timestamp
            nearestHour := time.Now().Truncate(time.Hour)
            forecastTimestampStr := fmt.Sprintf("%d", nearestHour.Unix())  

            dateForecastedFor := forecastTime.Format("2006-01-02 15:04:05")
            
            // Create the data field with raw values instead of type annotations
            dataMap := map[string]interface{}{
                "swellPeriod":           sample["swellPeriod"].(float64),
                "waterTemperature":      sample["waterTemperature"].(float64),
                "surfMessiness":         sample["surfMessiness"].(string),
                "pressure":              sample["pressure"].(float64),
                "waveEnergy":            sample["waveEnergy"].(float64),
                "precipitation":         sample["precipitation"].(float64),
                "relativeWindDirection": sample["relativeWindDirection"].(string),
                "swellHeight":           sample["swellHeight"].(float64),
                "swellDirection":        sample["swellDirection"].(float64),
                "temperature":           sample["temperature"].(float64),
                "directionQuality":      sample["directionQuality"].(float64),
                "humidity":              sample["humidity"].(float64),
                "surfSize":              sample["surfSize"].(float64),
                "windDirection":         sample["windDirection"].(float64),
                "windSpeed":             sample["windSpeed"].(float64),
                "dateForecastedFor":     dateForecastedFor,
            }
                    
            forecast := map[string]interface{}{
                "spot_id":            spotID,
                "forecast_timestamp": forecastTimestampStr,
                "data":               dataMap,
                "generated_at":       generatedAtTimestampStr,
            }
            
            item, err := dynamodbattribute.MarshalMap(forecast)
            if err != nil {
                return fmt.Errorf("failed to marshal forecast: %w", err)
            }
            
            _, err = storage.DB.PutItem(&dynamodb.PutItemInput{
                TableName: aws.String("SpotForecastData"),
                Item:      item,
            })
            
            if err != nil {
                return fmt.Errorf("failed to put forecast item: %w", err)
            }
            
            // Create additional entries at 3-hour intervals for better forecast coverage
            // between the sample points
            if !isLastSample(forecastSamples, sample) {
                nextSample := findNextSample(forecastSamples, sample)
                
                // Generate intermediate points (only if hours between samples > 3)
                hourDiff := nextSample["hourOffset"].(int) - sample["hourOffset"].(int)
                if hourDiff > 3 {
                    steps := hourDiff/3 - 1
                    for step := 1; step <= steps; step++ {
                        // Create interpolated forecast
                        progress := float64(step*3) / float64(hourDiff)
                        stepHours := sample["hourOffset"].(int) + step*3
                        
                        stepTime := baseTime.Add(time.Duration(stepHours) * time.Hour)
                        stepTimestampStr := fmt.Sprintf("%d", stepTime.Unix())
                        stepDateForecastedFor := stepTime.Format("2006-01-02 15:04:05")
                        
                        // Linear interpolation between samples
                        interpolatedSwellHeight := interpolate(sample["swellHeight"].(float64), nextSample["swellHeight"].(float64), progress)
                        interpolatedSwellPeriod := interpolate(sample["swellPeriod"].(float64), nextSample["swellPeriod"].(float64), progress)
                        interpolatedWindSpeed := interpolate(sample["windSpeed"].(float64), nextSample["windSpeed"].(float64), progress)
                        interpolatedWindDirection := interpolateAngle(sample["windDirection"].(float64), nextSample["windDirection"].(float64), progress)
                        interpolatedSwellDirection := interpolateAngle(sample["swellDirection"].(float64), nextSample["swellDirection"].(float64), progress)
                        interpolatedTemperature := interpolate(sample["temperature"].(float64), nextSample["temperature"].(float64), progress)
                        interpolatedWaterTemperature := interpolate(sample["waterTemperature"].(float64), nextSample["waterTemperature"].(float64), progress)
                        interpolatedPressure := interpolate(sample["pressure"].(float64), nextSample["pressure"].(float64), progress)
                        interpolatedHumidity := interpolate(sample["humidity"].(float64), nextSample["humidity"].(float64), progress)
                        interpolatedWaveEnergy := interpolate(sample["waveEnergy"].(float64), nextSample["waveEnergy"].(float64), progress)
                        interpolatedDirectionQuality := interpolate(sample["directionQuality"].(float64), nextSample["directionQuality"].(float64), progress)
                        interpolatedSurfSize := interpolate(sample["surfSize"].(float64), nextSample["surfSize"].(float64), progress)
                        interpolatedPrecipitation := interpolate(sample["precipitation"].(float64), nextSample["precipitation"].(float64), progress)
                        
                        // Create the data field with raw values for interpolated data
                        interpDataMap := map[string]interface{}{
                            "swellPeriod":           interpolatedSwellPeriod,
                            "waterTemperature":      interpolatedWaterTemperature,
                            "surfMessiness":         sample["surfMessiness"].(string),
                            "pressure":              interpolatedPressure,
                            "waveEnergy":            interpolatedWaveEnergy,
                            "precipitation":         interpolatedPrecipitation,
                            "relativeWindDirection": sample["relativeWindDirection"].(string),
                            "swellHeight":           interpolatedSwellHeight,
                            "swellDirection":        interpolatedSwellDirection,
                            "temperature":           interpolatedTemperature,
                            "directionQuality":      interpolatedDirectionQuality,
                            "humidity":              interpolatedHumidity,
                            "surfSize":              interpolatedSurfSize,
                            "windDirection":         interpolatedWindDirection,
                            "windSpeed":             interpolatedWindSpeed,
                            "dateForecastedFor":     stepDateForecastedFor,
                        }
                        
                        
                        interpolatedForecast := map[string]interface{}{
                            "spot_id":            spotID,
                            "forecast_timestamp": stepTimestampStr,
                            "data":               interpDataMap,
                            "generated_at":       generatedAtTimestampStr,
                        }
                        
                        item, err := dynamodbattribute.MarshalMap(interpolatedForecast)
                        if err != nil {
                            return fmt.Errorf("failed to marshal interpolated forecast: %w", err)
                        }
                        
                        _, err = storage.DB.PutItem(&dynamodb.PutItemInput{
                            TableName: aws.String("SpotForecastData"),
                            Item:      item,
                        })
                        
                        if err != nil {
                            return fmt.Errorf("failed to put interpolated forecast item: %w", err)
                        }
                    }
                }
            }
        }
    }
    
    return nil
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
    buoys := []map[string]interface{}{
        {
            "region_buoy": "NorthAtlantic_M6",
            "Latitude": 52.986,
            "Longitude": -15.866,
            "Name": "M6",
        },
        {
            "region_buoy": "NorthAtlantic_M3",
            "Latitude": 51.217,
            "Longitude": -10.55,
            "Name": "M3",
        },
        {
            "region_buoy": "NorthAtlantic_M4",
            "Latitude": 54.99972222,
            "Longitude": -9.998888889,
            "Name": "M4",
        },
        {
            "region_buoy": "NorthAtlantic_Blackstones",
            "Latitude": 56.06194444,
            "Longitude": -7.056666667,
            "Name": "Blackstones",
        },
        {
            "region_buoy": "NorthAtlantic_M2",
            "Latitude": 53.48,
            "Longitude": -5.425,
            "Name": "M2",
        },
        {
            "region_buoy": "NorthAtlantic_M5",
            "Latitude": 51.69,
            "Longitude": -6.704,
            "Name": "M5",
        },
        {
            "region_buoy": "NorthAtlantic_WestHebrides",
            "Latitude": 57.29194444,
            "Longitude": -7.913888889,
            "Name": "West Hebrides",
        },
    }
    
    for _, buoy := range buoys {
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
    }
    
    log.Printf("Successfully seeded %d buoy locations", len(buoys))
    return nil
}

func seedBuoyMeasurements() error {
    // Sample M4 buoy data - we'll use a condensed subset for brevity
    now := time.Now().UTC().Truncate(time.Hour)
    measurementsTemplate := []map[string]interface{}{
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 9.531,
            "AtmosphericPressure": 995.496,
            "DewPoint": nil,
            "Gust": 37.228,
            "MaxHeight": 14.219,
            "MaxPeriod": 18.164,
            "MeanWaveDirection": 262,
            "name": "M4",
            "RelativeHumidity": 71.777,
            "Salinity": 35.9893,
            "SeaTemperature": 10.979,
            "SprTp": 98.438,
            "ThTp": 274.219,
            "WaveHeight": 8.32,
            "WavePeriod": 10.547,
            "WindDirection": 243,
            "WindSpeed": 27.437,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 10.166,
            "AtmosphericPressure": 997.729,
            "DewPoint": nil,
            "Gust": 35.976,
            "MaxHeight": 12.656,
            "MaxPeriod": 14.297,
            "MeanWaveDirection": 264,
            "name": "M4",
            "RelativeHumidity": 67.09,
            "Salinity": 35.9906,
            "SeaTemperature": 10.992,
            "SprTp": 80.156,
            "ThTp": 268.594,
            "WaveHeight": 8.203,
            "WavePeriod": 10.43,
            "WindDirection": 242,
            "WindSpeed": 28.462,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 9.043,
            "AtmosphericPressure": 1002.112,
            "DewPoint": nil,
            "Gust": 29.714,
            "MaxHeight": 10.625,
            "MaxPeriod": 14.297,
            "MeanWaveDirection": 257,
            "name": "M4",
            "RelativeHumidity": 72.07,
            "Salinity": 35.99434,
            "SeaTemperature": 11.019,
            "SprTp": 99.844,
            "ThTp": 260.156,
            "WaveHeight": 5.977,
            "WavePeriod": 9.258,
            "WindDirection": 264,
            "WindSpeed": 22.2,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 8.506,
            "AtmosphericPressure": 1002.856,
            "DewPoint": nil,
            "Gust": 24.705,
            "MaxHeight": 9.063,
            "MaxPeriod": 12.539,
            "MeanWaveDirection": 259,
            "name": "M4",
            "RelativeHumidity": 74.512,
            "Salinity": 35.99617,
            "SeaTemperature": 11.032,
            "SprTp": 116.719,
            "ThTp": 255.938,
            "WaveHeight": 5.156,
            "WavePeriod": 8.438,
            "WindDirection": 281,
            "WindSpeed": 19.126,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 8.945,
            "AtmosphericPressure": 1003.125,
            "DewPoint": nil,
            "Gust": 35.407,
            "MaxHeight": 9.375,
            "MaxPeriod": 15.352,
            "MeanWaveDirection": 257,
            "name": "M4",
            "RelativeHumidity": 71.582,
            "Salinity": 35.99792,
            "SeaTemperature": 11.03,
            "SprTp": 168.75,
            "ThTp": 262.969,
            "WaveHeight": 5.156,
            "WavePeriod": 8.32,
            "WindDirection": 260,
            "WindSpeed": 23.794,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 8.896,
            "AtmosphericPressure": 1004.041,
            "DewPoint": nil,
            "Gust": 31.08,
            "MaxHeight": 8.438,
            "MaxPeriod": 12.539,
            "MeanWaveDirection": 263,
            "name": "M4",
            "RelativeHumidity": 75.293,
            "Salinity": 35.99442,
            "SeaTemperature": 11.027,
            "SprTp": 132.188,
            "ThTp": 260.156,
            "WaveHeight": 5.742,
            "WavePeriod": 8.906,
            "WindDirection": 271,
            "WindSpeed": 22.997,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 9.775,
            "AtmosphericPressure": 1008.569,
            "DewPoint": nil,
            "Gust": 31.422,
            "MaxHeight": 5.469,
            "MaxPeriod": 9.141,
            "MeanWaveDirection": 345,
            "name": "M4",
            "RelativeHumidity": 89.355,
            "Salinity": 35.92728,
            "SeaTemperature": 11.118,
            "SprTp": 118.125,
            "ThTp": 337.5,
            "WaveHeight": 3.164,
            "WavePeriod": 6.328,
            "WindDirection": 22,
            "WindSpeed": 24.136,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 9.922,
            "AtmosphericPressure": 1008.765,
            "DewPoint": nil,
            "Gust": 33.585,
            "MaxHeight": 6.25,
            "MaxPeriod": 9.961,
            "MeanWaveDirection": 350,
            "name": "M4",
            "RelativeHumidity": 83.105,
            "Salinity": 35.92506,
            "SeaTemperature": 11.113,
            "SprTp": 151.875,
            "ThTp": 338.906,
            "WaveHeight": 3.398,
            "WavePeriod": 6.563,
            "WindDirection": 24,
            "WindSpeed": 23.225,
        },
        {
            "region_buoy": "Ireland_M4",
            "AirTemperature": 9.873,
            "AtmosphericPressure": 1008.752,
            "DewPoint": nil,
            "Gust": 33.357,
            "MaxHeight": 5.313,
            "MaxPeriod": 9.961,
            "MeanWaveDirection": 352,
            "name": "M4",
            "RelativeHumidity": 81.445,
            "Salinity": 35.92529,
            "SeaTemperature": 11.104,
            "SprTp": 109.688,
            "ThTp": 341.719,
            "WaveHeight": 3.633,
            "WavePeriod": 6.68,
            "WindDirection": 29,
            "WindSpeed": 23.908,
        },
    }

    buoys := []map[string]interface{}{
        {
            "region_buoy": "Ireland_M6",
            "name": "M6",
            "wave_height_factor": 1.35,  
            "wave_period_offset": 3.0,  
            "max_height_factor": 1.4,
        },
        {
            "region_buoy": "Ireland_M3",
            "name": "M3",
            // South-west of Ireland, also sees larger swells
            "wave_height_factor": 1.25,
            "wave_period_offset": 2.5,
            "max_height_factor": 1.3,
        },
        {
            "region_buoy": "Ireland_M4",
            "name": "M4",
            // Northwest coast, our baseline
            "wave_height_factor": 1.0,
            "wave_period_offset": 0.0,
            "max_height_factor": 1.0,
        },
        {
            "region_buoy": "Ireland_Blackstones",
            "name": "Blackstones",
            // Western approaches - sees large swells
            "wave_height_factor": 1.4,
            "wave_period_offset": 2.8,
            "max_height_factor": 1.45,
        },
        {
            "region_buoy": "Ireland_M2",
            "name": "M2",
            // East coast, much more sheltered
            "wave_height_factor": 0.6,
            "wave_period_offset": -1.0,
            "max_height_factor": 0.65,
        },
        {
            "region_buoy": "Ireland_M5",
            "name": "M5",
            // Southeast coast, somewhat sheltered
            "wave_height_factor": 0.75,
            "wave_period_offset": -0.5,
            "max_height_factor": 0.8,
        },
        {
            "region_buoy": "Ireland_WestHebrides",
            "name": "West Hebrides",
            // Further north, exposed to Atlantic swells
            "wave_height_factor": 1.2,
            "wave_period_offset": 1.5,
            "max_height_factor": 1.25,
        },
    }
    
    totalCount := 0
    
    // For each buoy, generate measurement data
    for _, buoy := range buoys {
        buoyMeasurements := make([]map[string]interface{}, len(measurementsTemplate))
        
        // Get the variation factors for this buoy
        waveHeightFactor := buoy["wave_height_factor"].(float64)
        wavePeriodOffset := buoy["wave_period_offset"].(float64)
        maxHeightFactor := buoy["max_height_factor"].(float64)
        
        for i, template := range measurementsTemplate {
            // Clone the template
            measurement := make(map[string]interface{})
            for k, v := range template {
                measurement[k] = v
            }
            
            // Add buoy-specific fields
            measurement["region_buoy"] = buoy["region_buoy"]
            measurement["name"] = buoy["name"]
            
            // Apply wave variations to make each buoy unique
            // Adjust wave period to be between 8-16 seconds
            basePeriod := measurement["WavePeriod"].(float64)
            adjustedPeriod := math.Max(8.0, math.Min(16.0, basePeriod + wavePeriodOffset))
            measurement["WavePeriod"] = adjustedPeriod
            
            // Also adjust MaxPeriod to be consistent
            baseMaxPeriod := measurement["MaxPeriod"].(float64)
            adjustedMaxPeriod := math.Max(10.0, math.Min(18.0, baseMaxPeriod + wavePeriodOffset))
            measurement["MaxPeriod"] = adjustedMaxPeriod
            
            // Adjust wave height to be between 0.2-5m
            baseHeight := measurement["WaveHeight"].(float64)
            adjustedHeight := math.Max(0.2, math.Min(5.0, baseHeight * waveHeightFactor))
            measurement["WaveHeight"] = adjustedHeight
            
            // Also adjust MaxHeight to be consistent
            baseMaxHeight := measurement["MaxHeight"].(float64)
            adjustedMaxHeight := math.Max(0.5, math.Min(8.0, baseMaxHeight * maxHeightFactor))
            measurement["MaxHeight"] = adjustedMaxHeight
            
            // Set the timestamp to be relative to now, with most recent being last
            // Reverse the order so the most recent entry is last in the array
            hourOffset := len(measurementsTemplate) - 1 - i
            timestamp := now.Add(time.Duration(-hourOffset) * time.Hour)
            measurement["dataDateTime"] = timestamp.Format(time.RFC3339)
            
            buoyMeasurements[i] = measurement
        }
        
        // For each measurement, create an entry in DynamoDB
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
        
        log.Printf("Successfully seeded %d measurements for buoy %s (period range: %.1f-%.1fs, height range: %.1f-%.1fm)", 
            len(buoyMeasurements), 
            buoy["name"], 
            buoyMeasurements[0]["WavePeriod"].(float64),
            buoyMeasurements[len(buoyMeasurements)-1]["WavePeriod"].(float64),
            buoyMeasurements[0]["WaveHeight"].(float64),
            buoyMeasurements[len(buoyMeasurements)-1]["WaveHeight"].(float64))
        
        totalCount += len(buoyMeasurements)
    }
    
    log.Printf("Successfully seeded %d total buoy measurements", totalCount)
    return nil
}

// Helper function to check if a sample is the last one in the array
func isLastSample(samples []map[string]interface{}, current map[string]interface{}) bool {
    lastSample := samples[len(samples)-1]
    return current["hourOffset"] == lastSample["hourOffset"]
}

// Helper function to find the next sample after the current one
func findNextSample(samples []map[string]interface{}, current map[string]interface{}) map[string]interface{} {
    currentOffset := current["hourOffset"].(int)
    var nextSample map[string]interface{}
    minOffsetDiff := math.MaxInt32
    
    // Find the sample with the smallest hour offset greater than the current sample
    for _, sample := range samples {
        sampleOffset := sample["hourOffset"].(int)
        if sampleOffset > currentOffset && sampleOffset-currentOffset < minOffsetDiff {
            minOffsetDiff = sampleOffset - currentOffset
            nextSample = sample
        }
    }
    
    // If no next sample found (should not happen in normal operation), return the first sample
    if nextSample == nil {
        return samples[0]
    }
    
    return nextSample
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