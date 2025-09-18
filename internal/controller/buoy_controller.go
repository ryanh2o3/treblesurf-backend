package controller

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"treblesurf-backend/internal/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gin-gonic/gin"
)

func BuoyLocationInfo(c *gin.Context) {
    input := &dynamodb.ScanInput{
        TableName: aws.String("BuoyLocations"),
    }

    result, err := DB.Scan(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var locations []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &locations)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, locations)
}

func IndividualBuoyLocationInfo(c *gin.Context) {
    regionName := c.Query("region")
    buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")        
    input := &dynamodb.QueryInput{
        TableName: aws.String("BuoyLocations"),
        KeyConditionExpression: aws.String("region_buoy = :region_buoy"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":region_buoy": {
                S: aws.String(fmt.Sprintf("%s_%s", regionName, buoyName)),
            },
        },
    }

    result, err := DB.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "No buoy location found"})
        return
    }

    var buoy map[string]interface{}
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &buoy)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, buoy)
}

func GetLiveBuoyData(c *gin.Context) {
    buoys := []string{"M4", "Blackstones", "West Hebrides", "M2", "M3", "M5", "M6"}
    var buoyData []map[string]interface{}

    for _, buoy := range buoys {
        data := getBuoyData(buoy, "string")
        buoyData = append(buoyData, data)
    }

    c.JSON(http.StatusOK, buoyData)
}
// Example handler function to use the above function
func GetBuoyDataRange(c *gin.Context) {
    buoyName := c.Query("buoyName")
    startTimeStr := c.Query("startTime") // expected format: 2006-01-02T15:00:00Z
    endTimeStr := c.Query("endTime")     // expected format: 2006-01-02T15:00:00Z

    startTime, err := time.Parse("2006-01-02T15:00:00Z", startTimeStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format"})
        return
    }

    endTime, err := time.Parse("2006-01-02T15:00:00Z", endTimeStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format"})
        return
    }

    data, err := getBuoyDataRange(buoyName, startTime, endTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, data)
}

func GetSingleBuoyData(c *gin.Context) {
    // buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")    
    var data map[string]interface{}
    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    for i := 0; i < 12; i++ {
        searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
        dateStr := searchTime.UTC().Format("2006-01-02T15:00:00Z")
        data = getBuoyData(c.Query("buoyName"), dateStr)
        if data != nil {
            break
        }
    }

    c.JSON(http.StatusOK, data)
}

func GetLast24HoursBuoyData(c *gin.Context) {
    // buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")    
    // Calculate time range
    endTime := time.Now().UTC()
    startTime := endTime.AddDate(0, 0, -1) // 7 days ago
    
    // Get the data range
    data, err := getBuoyDataRange(c.Query("buoyName"), startTime, endTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    if len(data) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No data found for the last week"})
        return
    }
    
    c.JSON(http.StatusOK, data)
}

func GetMultipleBuoyData(c *gin.Context) {
    buoysStr := c.Query("buoys")
    buoys := strings.Split(buoysStr, ",")
    var values []map[string]interface{}

    for _, buoy := range buoys {
        var data map[string]interface{}
        // Start from current time rounded down to the nearest hour
        now := time.Now()
        currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
    
        for i := 0; i < 12; i++ {
            searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
            dateStr := searchTime.UTC().Format("2006-01-02T15:00:00Z")
            // buoyName := strings.ReplaceAll(buoy, " ", "")
            data = getBuoyData(buoy, dateStr)
            if data != nil {
                break
            }
        }
        if(data != nil){
        values = append(values, data)
        }
    }

    c.JSON(http.StatusOK, values)
}

func getBuoyDataRange(buoyName string, startTime, endTime time.Time) ([]map[string]interface{}, error) {
    startStr := startTime.UTC().Format("2006-01-02T15:00:00Z")
    endStr := endTime.UTC().Format("2006-01-02T15:00:00Z")
    input := &dynamodb.QueryInput{
        TableName: aws.String("BuoyData"),
        KeyConditionExpression: aws.String("region_buoy = :rb AND dataDateTime BETWEEN :start AND :end"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":rb": {
                S: aws.String(fmt.Sprintf("Ireland_%s", buoyName)),
            },
            ":start": {
                S: aws.String(startStr),
            },
            ":end": {
                S: aws.String(endStr),
            },
        },
        ScanIndexForward: aws.Bool(true), // true for ascending order by time
    }

    var allItems []map[string]interface{}
    for {
        result, err := DB.Query(input)
        if err != nil {
            return nil, fmt.Errorf("error querying buoy data: %v", err)
        }

        var items []map[string]interface{}
        if err := dynamodbattribute.UnmarshalListOfMaps(result.Items, &items); err != nil {
            return nil, fmt.Errorf("error unmarshalling buoy data: %v", err)
        }
        
        allItems = append(allItems, items...)

        // Handle pagination
        if result.LastEvaluatedKey == nil {
            break
        }
        input.ExclusiveStartKey = result.LastEvaluatedKey
    }

    return allItems, nil
}

func getBuoyData(buoyName string, dateStr string) map[string]interface{} {
    var buoyData map[string]interface{}

    input := &dynamodb.QueryInput{
        TableName: aws.String("BuoyData"),
        KeyConditionExpression: aws.String("region_buoy = :rb AND dataDateTime = :dt"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":rb": {
                S: aws.String(fmt.Sprintf("Ireland_%s", buoyName)),
            },
            ":dt": {
                S: aws.String(dateStr),
            },
        },
        ScanIndexForward: aws.Bool(false), // Get most recent first
        Limit:           aws.Int64(1),     // We only want the most recent reading
    }

    result, err := DB.Query(input)
    if err != nil {
        log.Printf("Error querying buoy data: %v", err)
        return nil
    }

    if len(result.Items) == 0 {
        return nil
    }

    
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &buoyData)
    if err != nil {
        log.Printf("Error unmarshalling buoy data: %v", err)
        return nil
    }
    return buoyData
    
}

func GetRegionBuoys(c *gin.Context){
    regionName := c.Query("region")
    var buoys []model.Buoy



    input := &dynamodb.ScanInput{
		TableName: aws.String("BuoyLocations"),
		FilterExpression: aws.String("begins_with(region_buoy, :region)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":region": {
				S: aws.String(fmt.Sprintf("%s_", regionName)),
			},
		},
	}

	result, err := DB.Scan(input)
	if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to query"})
        return
	}

	if len(result.Items) == 0 {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "no buoys found for this region"})
        return
	}

    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &buoys)
    if err != nil {
        log.Printf("Error unmarshalling buoy data: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to find buoys for this region"})
        return
    }
    c.JSON(http.StatusOK, buoys)
}