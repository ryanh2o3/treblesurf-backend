package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
)

var ( 
    db *dynamodb.DynamoDB
    s3Client *s3.S3
    rekognitionClient *rekognition.Rekognition
)

func init() {
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-west-1"),
    }))
    db = dynamodb.New(sess)
    s3Client = s3.New(sess)
    rekognitionClient = rekognition.New(sess)
}

type Location struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type Forecast struct {
    DateForecastedFor string `json:"dateForecastedFor"`
}

type ReportWithImage struct {
    Country        string `json:"country"`
    Region         string `json:"region"`
    Spot           string `json:"spot"`
    SurfSize       string `json:"surfSize"`
    WindAmount     string `json:"windAmount"`
    WindDirection  string `json:"windDirection"`
    Consistency    string `json:"consistency"`
    Quality        string `json:"quality"`
    Messiness      string `json:"messiness"`
    ImageData      string `json:"imageData"` // Base64 encoded image
}

func GetRegions(c *gin.Context) {
    countryName := c.Query("country")

    input := &dynamodb.ScanInput{
        TableName: aws.String("LocationData"),
    }

    result, err := db.Scan(input)
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

    var regions []string
    for _, location := range locations {
        log.Printf("Location: %v", location)
        parts := strings.Split(location["country_region_spot"].(string), "/")
        if parts[0] == countryName {
            region := parts[1]
            if !contains(regions, region) {
                regions = append(regions, region)
            }
        }
    }

    c.JSON(http.StatusOK, regions)
}


func GetCoordinates(c *gin.Context) {
    spotName := c.Query("spot")
    regionName := c.Query("region")
    countryName := c.Query("country")

    input := &dynamodb.QueryInput{
        TableName: aws.String("LocationData"),
        KeyConditionExpression: aws.String("country_region_spot = :location"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":location": {
                S: aws.String(fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)),
            },
        },
    }

    result, err := db.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No coordinates found"})
        return
    }

    var location map[string]interface{}
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &location)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    coordinates := []float64{
        location["Latitude"].(float64),
        location["Longitude"].(float64),
    }

    c.JSON(http.StatusOK, coordinates)
}

func GetLocationInfo(c *gin.Context) {
    spotName := c.Query("spot")
    regionName := c.Query("region")
    countryName := c.Query("country")

    input := &dynamodb.ScanInput{
        TableName: aws.String("LocationData"),
        FilterExpression: aws.String("country_region_spot = :location"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":location": {
                S: aws.String(fmt.Sprintf("%s/%s/%s", countryName, regionName, spotName)),
            },
        },
    }

    result, err := db.Scan(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No location found"})
        return
    }

    var location map[string]interface{}
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &location)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, location)
}

func BuoyLocationInfo(c *gin.Context) {
    input := &dynamodb.ScanInput{
        TableName: aws.String("BuoyLocations"),
    }

    result, err := db.Scan(input)
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
    buoyName := c.Query("buoyName")

    input := &dynamodb.QueryInput{
        TableName: aws.String("BuoyLocations"),
        KeyConditionExpression: aws.String("region_buoy = :region_buoy"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":region_buoy": {
                S: aws.String(fmt.Sprintf("%s_%s", regionName, buoyName)),
            },
        },
    }

    result, err := db.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No buoy location found"})
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

func GetSpots(c *gin.Context) {
    regionName := c.Query("region")
    countryName := c.Query("country")

    input := &dynamodb.ScanInput{
        TableName: aws.String("LocationData"),
        FilterExpression: aws.String("begins_with(country_region_spot, :location)"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":location": {
                S: aws.String(fmt.Sprintf("%s/%s/", countryName, regionName)),
            },
        },
    }

    result, err := db.Scan(input)
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

    var spots []string
    for _, location := range locations {
        parts := strings.Split(location["country_region_spot"].(string), "/")
        if len(parts) == 3 {
            spots = append(spots, parts[2])
        }
    }

    c.JSON(http.StatusOK, spots)
}

func GetSpotForecast(c *gin.Context) {
    spotName := c.Query("spot")
    regionName := c.Query("region")
    countryName := c.Query("country")

    forecast, err := queryForecastByDateTime(spotName, regionName, countryName, nil)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    if forecast != nil {
        c.JSON(http.StatusOK, forecast)
        return
    }
    

    c.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func queryForecastByDateTime(spotName, regionName, countryName string, limit *int64) ([]map[string]interface{}, error) {
    spotId := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
    currentEpoch := time.Now().Unix()
    
    input := &dynamodb.QueryInput{
        TableName: aws.String("SpotForecastData"),
        KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":spot_id": {
                S: aws.String(spotId),
            },
            ":current_time": {
                S: aws.String(fmt.Sprintf("%d", currentEpoch)),
            },
        },
        ScanIndexForward: aws.Bool(true), // Sort by forecast_timestamp in ascending order
        
    }
    if limit != nil {
        input.Limit = limit
    }

    result, err := db.Query(input)
    if err != nil {
        return nil, err
    }

    var forecasts []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
    if err != nil {
        return nil, err
    }

    return forecasts, nil
}

func queryMultipleSpotForecasts(spotIds []string, limit *int64) ([][]map[string]interface{}, error) {
    currentEpoch := time.Now().Unix()
    var allForecasts [][]map[string]interface{}

    // Query each spot ID separately since BatchGetItem doesn't support range queries
    for _, spotId := range spotIds {
        input := &dynamodb.QueryInput{
            TableName: aws.String("SpotForecastData"),
            KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
            ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
                ":spot_id": {
                    S: aws.String(spotId),
                },
                ":current_time": {
                    S: aws.String(fmt.Sprintf("%d", currentEpoch)),
                },
            },
            ScanIndexForward: aws.Bool(true), // Sort by forecast_timestamp in ascending order
        }
        if limit != nil {
            input.Limit = limit
        }

        result, err := db.Query(input)
        if err != nil {
            return nil, err
        }

        var forecasts []map[string]interface{}
        err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
        if err != nil {
            return nil, err
        }

        allForecasts = append(allForecasts, forecasts)
    }

    return allForecasts, nil
}

// func queryForecastByDateTimeLast(spotName, regionName, countryName, dateTime string) (map[string]interface{}, error) {
//     print(dateTime)
//     input := &dynamodb.QueryInput{
//         TableName: aws.String("SurfSpotForecastData"),
//         KeyConditionExpression: aws.String("forecastDate = :dateTime AND begins_with(country_region_spot, :location)"),
//         ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
//             ":dateTime": {
//                 S: aws.String(dateTime),
//             },
//             ":location": {
//                 S: aws.String(fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)),
//             },
//         },
//         ScanIndexForward: aws.Bool(true), 
//     }

//     result, err := db.Query(input)
//     if err != nil {
//         return nil, err
//     }

//     if len(result.Items) == 0 {
//         return nil, nil
//     }

//     var forecasts map[string]interface{}
//     err = dynamodbattribute.UnmarshalMap(result.Items[0], &forecasts)
//     if err != nil {
//         return nil, err
//     }

//     return forecasts, nil
// }


func GetListSpotsForecast(c *gin.Context) {
    spotsStr := c.Query("spots")
    spots := strings.Split(spotsStr, ",")
    regionName := c.Query("region")
    countryName := c.Query("country")
    
    var spotIds []string
    for _, spot := range spots {
        spotIds = append(spotIds, fmt.Sprintf("%s#%s#%s", countryName, regionName, spot))
    }

    log.Print(spotIds)

    forecasts, err := queryMultipleSpotForecasts(spotIds, aws.Int64(72))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(forecasts) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No forecasts found"})
        return
    }

    c.JSON(http.StatusOK, forecasts)
}

func GetRegionForecast(c *gin.Context) {
    regionName := c.Query("region")
    countryName := c.Query("country")
    forecastDate := time.Now().Format("2006-01-02")

    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfSpotForecastData"),
        KeyConditionExpression: aws.String("ForecastDate = :date AND begins_with(country_region_spot, :location)"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":date": {
                S: aws.String(forecastDate),
            },
            ":location": {
                S: aws.String(fmt.Sprintf("%s_%s_", countryName, regionName)),
            },
        },
        ScanIndexForward: aws.Bool(false),
    }

    result, err := db.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No forecast found"})
        return
    }

    var forecasts []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Sort the forecasts by country, region, and spot
    sort.Slice(forecasts, func(i, j int) bool {
        return forecasts[i]["country_region_spot"].(string) < forecasts[j]["country_region_spot"].(string)
    })

    c.JSON(http.StatusOK, forecasts)
}

func GetCurrentWeather(c *gin.Context) {
    spotName := c.Query("spot")
    regionName := c.Query("region")
    countryName := c.Query("country")

    forecast, err := queryForecastByDateTime(spotName, regionName, countryName, aws.Int64(1))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    if forecast != nil {
        c.JSON(http.StatusOK, forecast)
        return
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}


func getCurrentTides(locationName string) []map[string]interface{} {

    today := time.Now().Format("2006-01-02")
    yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
    tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

    queryToday := getTides(locationName, today)
    queryYesterday := getTides(locationName, yesterday)
    queryTomorrow := getTides(locationName, tomorrow)

    query := mergeMaps(queryYesterday, queryToday, queryTomorrow)
    var result []map[string]interface{}
    for _, v := range query {
        result = append(result, v.(map[string]interface{}))
    }
    return result
}

func GetBeforeAfterTides(c *gin.Context) {
    locationName := c.Query("locationName")

    tides := getCurrentTides(locationName)
    now := time.Now()

    var prevTide, nextTide map[string]interface{}
    for _, tide := range tides {
        tideTime, _ := time.Parse("2006-01-02 15:04:05", tide["time"].(string))
        if tideTime.Before(now) && (prevTide == nil || tideTime.After(prevTide["time"].(time.Time))) {
            prevTide = tide
        }
        if tideTime.After(now) && (nextTide == nil || tideTime.Before(nextTide["time"].(time.Time))) {
            nextTide = tide
        }
    }

    c.JSON(http.StatusOK, gin.H{"prevTide": prevTide, "nextTide": nextTide})
}

func GetDayTides(c *gin.Context) {
    locationName := c.Query("locationName")
    startDay := c.Param("startDay")

    start, _ := time.Parse("2006-01-02", startDay)
    end := start.AddDate(0, 0, 10)

    tideData := make(map[string]interface{})
    for start.Before(end) {
        day := start.Format("2006-01-02")
        tideData[day] = getTides(locationName, day)
        start = start.AddDate(0, 0, 1)
    }

    c.JSON(http.StatusOK, tideData)
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
    buoyName := c.Query("buoyName")
    var data map[string]interface{}
    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    for i := 0; i < 12; i++ {
        searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
        dateStr := searchTime.UTC().Format("2006-01-02T15:00:00Z")
        data = getBuoyData(buoyName, dateStr)
        if data != nil {
            break
        }
    }

    c.JSON(http.StatusOK, data)
}

func GetLast24HoursBuoyData(c *gin.Context) {
    buoyName := c.Query("buoyName")
    
    // Calculate time range
    endTime := time.Now().UTC()
    startTime := endTime.AddDate(0, 0, -1) // 7 days ago
    
    // Get the data range
    data, err := getBuoyDataRange(buoyName, startTime, endTime)
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

func SubmitCurrentSurfReport(c *gin.Context) {
    var report ReportWithImage
    if err := c.BindJSON(&report); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
	}

    // if !isValidSwellSize(report["surfSize"]) || !isValidWindAmount(report["windAmount"]) || !isValidWindDirection(report["windDirection"]) || !isValidSurfConditions(report["surfConditions"]) || !isValidSurfDifficulty(report["surfDifficulty"]) {
    //     c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report data"})
    //     return
    // }

    currentTime := time.Now()

    countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)

    dateReported := fmt.Sprintf("%s_%s", currentTime, email)
    // Create the DynamoDB item
    item := map[string]*dynamodb.AttributeValue{
        "country_region_spot": {
            S: aws.String(countryRegionSpot),
        },
        "dateReported": {
            S: aws.String(dateReported),
        },
        "SurfSize": {
            S: aws.String(report.SurfSize),
        },
        "WindAmount": {
            S: aws.String(report.WindAmount),
        },
        "WindDirection": {
            S: aws.String(report.WindDirection),
        },
        "Consistency": {
            S: aws.String(report.Consistency),
        },
        "Quality": {
            S: aws.String(report.Quality),
        },
        "Messiness": {
            S: aws.String(report.Messiness),
        },
        "UserEmail": {
            S: aws.String(email.(string)),
        },
        "Time": {
            S: aws.String(currentTime.String()),
        },
    }

    // Process image if provided
    if report.ImageData != "" {
        // Extract base64 data
        base64String := report.ImageData
        // Handle data URIs by removing the prefix
        if strings.HasPrefix(base64String, "data:") {
            // Find the comma that separates the header from the data
            commaIndex := strings.Index(base64String, ",")
            if commaIndex != -1 {
                base64String = base64String[commaIndex+1:]
            }
        }
        imageData, err := base64.StdEncoding.DecodeString(base64String)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image data"})
            return
        }

        // Validate image using Rekognition
        valid, err := validateImageWithRekognition(imageData)
        if err != nil {
            log.Printf("Rekognition error: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate image"})
            return
        }

        if valid {
            // Upload to S3
            imageKey := fmt.Sprintf("surf-reports/%s/%s.jpg", countryRegionSpot, dateReported)
            s3Key, err := uploadImageToS3(imageData, imageKey)
            if err != nil {
                log.Printf("S3 upload error: %v", err)
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
                return
            }

            // Store just the S3 key in DynamoDB
            item["ImageKey"] = &dynamodb.AttributeValue{
                S: aws.String(s3Key),
            }
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Image validation failed"})
            return
        }
    }

    // Insert into DynamoDB
    input := &dynamodb.PutItemInput{
        TableName: aws.String("SurfReports"),
        Item: item,
    }


    _, err := db.PutItem(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

func validateImageWithRekognition(imageData []byte) (bool, error) {
    input := &rekognition.DetectLabelsInput{
        Image: &rekognition.Image{
            Bytes: imageData,
        },
        MinConfidence: aws.Float64(90.0),
    }

    result, err := rekognitionClient.DetectLabels(input)
    if err != nil {
        return false, err
    }

    validLabels := []string{"Sea", "Water", "Sea Waves", "Beach", "Coast"}
    for _, label := range result.Labels {
        for _, validLabel := range validLabels {
            if strings.EqualFold(*label.Name, validLabel) {
                return true, nil
            }
        }
    }

    return false, nil
}

func uploadImageToS3(imageData []byte, key string) (string, error) {
    bucketName := "treblesurf-images" // Replace with your bucket name
    
    _, err := s3Client.PutObject(&s3.PutObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(key),
        Body:   aws.ReadSeekCloser(strings.NewReader(string(imageData))),
        ContentType: aws.String("image/jpeg"),
    })
    if err != nil {
        return "", err
    }

    // Return the URL for the uploaded image
    return key, nil
}


func RetrieveTodaysSurfReports(c *gin.Context) {
    countryName := c.Query("country")
    regionName := c.Query("region")
    spotName := c.Query("spot")

    countryRegionSpot := fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)

    todayPrefix := time.Now().Format("2006-01-02")
    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfReports"),
        KeyConditionExpression: aws.String("country_region_spot = :crs AND begins_with(dateReported, :today)"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":crs":   {S: aws.String(countryRegionSpot)},
            ":date":    {S: aws.String(todayPrefix)},
        },
    }

    result, err := db.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var reports []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &reports)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // For each report with an image key, add the base64 image data if requested
    includeImages := c.Query("includeImages") == "true"
    if includeImages {
        for i, report := range reports {
            if imageKey, ok := report["ImageKey"].(string); ok && imageKey != "" {
                // Get the image from S3
                image, err := getImageFromS3(imageKey)
                if err == nil {
                    // Add base64-encoded image to the report
                    reports[i]["ImageData"] = base64.StdEncoding.EncodeToString(image)
                }
            }
        }
    }

    c.JSON(http.StatusOK, reports)
}

func getImageFromS3(key string) ([]byte, error) {
    result, err := s3Client.GetObject(&s3.GetObjectInput{
        Bucket: aws.String("treblesurf-images"),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()
    
    return io.ReadAll(result.Body)
}

func GetReportImage(c *gin.Context) {
    // Get the image key from the query parameter
    imageKey := c.Query("key")
    if imageKey == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Image key is required"})
        return
    }
    
    // Get the image from S3
    result, err := s3Client.GetObject(&s3.GetObjectInput{
        Bucket: aws.String("treblesurf-images"),
        Key:    aws.String(imageKey),
    })
    if err != nil {
        log.Printf("Error getting image from S3: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve image"})
        return
    }
    defer result.Body.Close()
    
    // Read the image data
    imageData, err := io.ReadAll(result.Body)
    if err != nil {
        log.Printf("Error reading image data: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image data"})
        return
    }
    
    // Convert to base64
    base64Data := base64.StdEncoding.EncodeToString(imageData)
    
    // Return the base64-encoded image
    c.JSON(http.StatusOK, gin.H{
        "imageData": base64Data,
        "contentType": *result.ContentType,
    })
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



// Helper functions
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func getTides(locationName, date string) map[string]interface{} {
    // Implement the logic to get tides from DynamoDB
    return nil
}

func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
    merged := make(map[string]interface{})
    for _, m := range maps {
        for k, v := range m {
            merged[k] = v
        }
    }
    return merged
}

func getBuoyData(buoyName string, dateStr string) map[string]interface{} {
    var buoyData map[string]interface{}
    print(dateStr)
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

    result, err := db.Query(input)
    if err != nil {
        log.Printf("Error querying buoy data: %v", err)
        return nil
    }

    if len(result.Items) == 0 {
        log.Printf("No buoy data found for %s", buoyName)
        return nil
    }

    
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &buoyData)
    if err != nil {
        log.Printf("Error unmarshalling buoy data: %v", err)
        return nil
    }
    return buoyData
    
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
        result, err := db.Query(input)
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



func getBuoyDataLast24Hours(buoyName string) map[string]interface{} {
    // Implement the logic to get last 24 hours buoy data from DynamoDB
    return nil
}

func isValidSwellSize(swellSize string) bool {
    validSizes := []string{"flat", "0-0.5", "0.5-1", "1-1.5", "1.5-2.5", "2.5+"}
    return contains(validSizes, swellSize)
}

func isValidWindAmount(windAmount string) bool {
    validAmounts := []string{"calm", "light", "moderate", "strong"}
    return contains(validAmounts, windAmount)
}

func isValidWindDirection(windDirection string) bool {
    validDirections := []string{"glassy", "offshore", "cross", "onshore"}
    return contains(validDirections, windDirection)
}

func isValidSurfConditions(surfConditions string) bool {
    validConditions := []string{"glassy", "clean", "messy", "okay"}
    return contains(validConditions, surfConditions)
}

func isValidSurfDifficulty(surfDifficulty string) bool {
    validDifficulties := []string{"lulls", "consistent", "relentless"}
    return contains(validDifficulties, surfDifficulty)
}


func DeleteMyAccount(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

     // First check if the user exists
     user, err := getUserByEmail(email.(string))
     if err != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
         return
     }
 
     if user == nil {
         c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
         return
     }

	// Delete the user account
    input := &dynamodb.DeleteItemInput{
        TableName: aws.String("Users"),
        Key: map[string]*dynamodb.AttributeValue{
            "email": {
                S: aws.String(email.(string)),
            },
        },
    }

    _, err = db.DeleteItem(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
        return
    }

    // Here you could add additional cleanup for user data if needed
    // For example:
    // - Delete user preferences
    // - Delete saved spots
    // - Remove user reports/contributions

    c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

func GetUserTheme(c *gin.Context) {
    email, exists := c.Get("email")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

     // First check if the user exists
     user, err := getUserByEmail(email.(string))
     if err != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
         return
     }
 
     if user == nil {
         c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
         return
     }

    theme := user.Theme
    c.JSON(http.StatusOK, gin.H{"theme": theme})
}

func SetUserTheme(c *gin.Context) {
    email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
	}

     // First check if the user exists
     user, err := getUserByEmail(email.(string))
     if err != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
        return
        }
 
     if user == nil {
         c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
         return
     }
     theme := c.Query("theme")
     if theme == "" {
         c.JSON(http.StatusBadRequest, gin.H{"error": "Theme is required"})
         return
     }
 
     input := &dynamodb.UpdateItemInput{
         TableName: aws.String("Users"),
         Key: map[string]*dynamodb.AttributeValue{
             "email": {
                 S: aws.String(email.(string)),
             },
         },
         UpdateExpression: aws.String("SET theme = :theme"),
         ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
             ":theme": {
                 S: aws.String(theme),
             },
         },
     }
 
     _, err = db.UpdateItem(input)
     if err != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme"})
         return
     }
 
     c.JSON(http.StatusOK, gin.H{"message": "Theme updated successfully"})
}