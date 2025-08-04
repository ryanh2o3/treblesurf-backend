package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/kinesisvideo"
	"github.com/aws/aws-sdk-go/service/kinesisvideoarchivedmedia"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
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

type LocationInfo struct {
    BeachDirection      int     `json:"BeachDirection"`
    Elevation           int     `json:"Elevation"`
    IdealSwellDirection string  `json:"IdealSwellDirection"`
    Image               string  `json:"Image"`
    Latitude            float64 `json:"Latitude"`
    Longitude           float64 `json:"Longitude"`
    Type                string  `json:"Type"`
    CountryRegionSpot   string  `json:"country_region_spot"`
    ImageString        string  `json:"ImageString"`
}

func GetLocationInfo(c *gin.Context) {
    spotName := c.Query("spot")
    regionName := c.Query("region")
    countryName := c.Query("country")

    input := &dynamodb.QueryInput{
        TableName: aws.String("LocationData"),
        KeyConditionExpression: aws.String("country_region_spot = :location"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":location": {
                S: aws.String(fmt.Sprintf("%s/%s/%s", countryName, regionName, spotName)),
            },
        },
    }

    result, err := db.Query(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(result.Items) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No location found"})
        return
    }

    var location LocationInfo
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &location)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if location.Image != "" {
        imageData, err := getImageFromS3(location.Image + ".jpg")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve image"})
            return
        }
        location.ImageString = base64.StdEncoding.EncodeToString(imageData)
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

    c.JSON(http.StatusOK, locations)
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
    log.Print("start of submit report")
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
    user, err2 := getUserByEmail(email.(string))
     if err2 != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
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
        "Reporter": {
            S: aws.String(user.GivenName),
        },
    }
    var s3KeyReport = ""

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

        // Extract time from EXIF data if available
        exifData, err := exif.Decode(bytes.NewReader(imageData))
        if err == nil {
            if dateTime, err := exifData.DateTime(); err == nil {
                currentTime = dateTime
            }
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
            imageKey := fmt.Sprintf(
                "surf-reports/%s/%s_%s.jpg",
                countryRegionSpot,
                currentTime.UTC().Format("2006-01-02T15:04:05Z"),
                email,
            )
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
            s3KeyReport = s3Key
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Image validation failed"})
            return
        }
    }
    item["Time"] = &dynamodb.AttributeValue{
        S: aws.String(currentTime.String()),
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
    log.Print("done putting")

    message := map[string]interface{}{
    "action": "new_report",
    "data": map[string]interface{}{
        "country":       report.Country,
        "region":        report.Region,
        "spot":          report.Spot,
        "quality":       report.Quality,
        "surfSize":      report.SurfSize,
        "windAmount":    report.WindAmount,
        "windDirection": report.WindDirection,
        "messiness":     report.Messiness,
        "consistency":   report.Consistency,
        "reporter":      user.GivenName,
        "imageKey":      s3KeyReport,
        "reportTime":    time.Now().Format(time.RFC3339),
    },
}

    subscribers, err := getSpotSubscribers(report.Country, report.Region, report.Spot)
    log.Print("subscribers", subscribers)
    if err != nil {
        log.Printf("Failed to get subscribers: %v", err)
    } else {
        // Broadcast to subscribers asynchronously
        go func() {
            err := BroadcastToUsers(subscribers, message) // Use your stage name
            if err != nil {
                log.Printf("Failed to broadcast message: %v", err)
            }
        }()
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

    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfReports"),
        KeyConditionExpression: aws.String("country_region_spot = :crs"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":crs":   {S: aws.String(countryRegionSpot)},
        },
        ScanIndexForward: aws.Bool(false), // Sort in descending order to get the latest reports
        Limit:            aws.Int64(1),   // Limit to the last 5 reports
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

// getSpotSubscribers returns a list of user IDs who are subscribed to a specific spot
func getSpotSubscribers(country, region, spot string) ([]string, error) {
    // Create the spot identifier
    spotIdentifier := fmt.Sprintf("%s/%s/%s", country, region, spot)
    
    // Query the SpotSubscriptions table
    input := &dynamodb.QueryInput{
        TableName: aws.String("SpotSubscriptions"),
        KeyConditionExpression: aws.String("spot_id = :spotId"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":spotId": {
                S: aws.String(spotIdentifier),
            },
        },
    }
    
    result, err := db.Query(input)
    if err != nil {
        return nil, fmt.Errorf("error querying subscriptions: %v", err)
    }
    
    // Extract user IDs from the results
    var subscribers []string
    for _, item := range result.Items {
        if userID, ok := item["user_id"]; ok && userID.S != nil {
            subscribers = append(subscribers, *userID.S)
        }
    }
    
    return subscribers, nil
}

func GetStreamingCredentials(c *gin.Context) {
    apiKey, exists := c.Get("apiKey")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
        return
    }

    key := apiKey.(*APIKey)

    // Create an STS client
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-west-1"),
    }))

    // Request temporary credentials with proper permissions
    stsClient := sts.New(sess)
    result, err := stsClient.AssumeRole(&sts.AssumeRoleInput{
        RoleArn:         aws.String("arn:aws:iam::759663378274:role/TreblesurfPiStreamingRole"),
        RoleSessionName: aws.String("device-stream-" + key.KeyID),
        DurationSeconds: aws.Int64(3600), // 1 hour
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "accessKey":    *result.Credentials.AccessKeyId,
        "secretKey":    *result.Credentials.SecretAccessKey,
        "sessionToken": *result.Credentials.SessionToken,
        "expiration":   result.Credentials.Expiration.Format(time.RFC3339),
    })
}

// GetStreamPlaybackURL generates a signed URL for viewing the stream
func GetStreamPlaybackURL(c *gin.Context) {
    // Only authenticated users can access this endpoint
    email, exists := c.Get("email")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
        return
    }
    fmt.Print(email)
    
    // You can add additional authorization checks here
    // For example, check if the user has permission to view this camera
    
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-west-1"),
    }))
    
    kvsClient := kinesisvideo.New(sess)
    
    getDataEndpointOutput, err := kvsClient.GetDataEndpoint(&kinesisvideo.GetDataEndpointInput{
        StreamName:              aws.String("treblesurf-webcam"),
        APIName:                 aws.String("GET_HLS_STREAMING_SESSION_URL"),
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    archiveClient := kinesisvideoarchivedmedia.New(sess, &aws.Config{
        Endpoint: getDataEndpointOutput.DataEndpoint,
    })
    
    hlsOutput, err := archiveClient.GetHLSStreamingSessionURL(&kinesisvideoarchivedmedia.GetHLSStreamingSessionURLInput{
        StreamName:      aws.String("treblesurf-webcam"),
        PlaybackMode:    aws.String("LIVE"), // Use "ON_DEMAND" for recorded content
        HLSFragmentSelector: &kinesisvideoarchivedmedia.HLSFragmentSelector{
            FragmentSelectorType: aws.String("SERVER_TIMESTAMP"),
        },
        ContainerFormat:  aws.String("FRAGMENTED_MP4"),
        DiscontinuityMode: aws.String("ALWAYS"),
        Expires:          aws.Int64(3600), // URL valid for 1 hour
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "hlsUrl": *hlsOutput.HLSStreamingSessionURL,
    })
}


// CreateAPIKeyHandler handles requests to create a new API key
func CreateAPIKeyHandler(c *gin.Context) {
    email, _ := c.Get("email")
    
    var request struct {
        Description string   `json:"description" binding:"required"`
        ExpiryDays  int      `json:"expiry_days"`
        Scopes      []string `json:"scopes" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }
    
    apiKey, err := GenerateAPIKey(
        request.Description,
        email.(string),
        request.ExpiryDays,
        request.Scopes,
    )
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
        return
    }
    
    err = storeAPIKey(apiKey)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store API key"})
        return
    }
    
    // Return the API key (this is the only time the client will see the full key)
    c.JSON(http.StatusCreated, gin.H{
        "message": "API key created successfully",
        "key": apiKey,
    })
}

// ListAPIKeysHandler returns all API keys (without their values)
func ListAPIKeysHandler(c *gin.Context) {
    input := &dynamodb.ScanInput{
        TableName: aws.String("ApiKeys"),
        ProjectionExpression: aws.String("key_id, description, created_by, created_at, expires_at, scopes"),
    }
    
    result, err := db.Scan(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
        return
    }
    
    var apiKeys []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &apiKeys)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse API keys"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "keys": apiKeys,
        "count": len(apiKeys),
    })
}

// RevokeAPIKeyHandler deletes an API key
func RevokeAPIKeyHandler(c *gin.Context) {
    keyID := c.Param("keyID")
    
    input := &dynamodb.DeleteItemInput{
        TableName: aws.String("ApiKeys"),
        Key: map[string]*dynamodb.AttributeValue{
            "key_id": {
                S: aws.String(keyID),
            },
        },
    }
    
    _, err := db.DeleteItem(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}

type StreamRequest struct {
    SpotID     string    `json:"spot_id"`
    RequestedBy string    `json:"requested_by"`
    RequestedAt time.Time `json:"requested_at"`
    Expiration  int64     `json:"expiration"`
}

func RequestStreamHandler(c *gin.Context) {
    email, exists := c.Get("email")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
        return
    }

    var request struct {
        SpotID string `json:"spot_id" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format. Spot ID is required."})
        return
    }

    now := time.Now()
    expiration := now.Add(5 * time.Minute).Unix()
    
    streamRequest := StreamRequest{
        SpotID:     request.SpotID,
        RequestedBy: email.(string),
        RequestedAt: now,
        Expiration:  expiration,
    }

    item, err := dynamodbattribute.MarshalMap(streamRequest)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
        return
    }

    _, err = db.PutItem(&dynamodb.PutItemInput{
        TableName: aws.String("StreamRequests"),
        Item:      item,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save stream request"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Stream requested successfully",
        "expires_at": now.Add(5 * time.Minute).Format(time.RFC3339),
    })
}

func CheckStreamRequestHandler(c *gin.Context) {
    spotID := c.Query("spot_id")
    if spotID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing spot_id parameter"})
        return
    }

    // Query DynamoDB for this spot ID
    result, err := db.GetItem(&dynamodb.GetItemInput{
        TableName: aws.String("StreamRequests"),
        Key: map[string]*dynamodb.AttributeValue{
            "spot_id": {S: aws.String(spotID)},
        },
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stream request status"})
        return
    }

    streamRequested := len(result.Item) > 0

    if streamRequested {
        var request StreamRequest
        if err := dynamodbattribute.UnmarshalMap(result.Item, &request); err == nil {
            if time.Now().Unix() > request.Expiration {
                streamRequested = false
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "stream_requested": streamRequested,
    })
}

type SpotSnapshot struct {
    SpotID     string    `json:"spot_id"`
    ImageKey   string    `json:"image_key"`
    Timestamp  time.Time `json:"timestamp"`
    UploadedAt time.Time `json:"uploaded_at"`
}

// UploadSnapshotHandler handles image uploads from devices
func UploadSnapshotHandler(c *gin.Context) {    
    // Get the spot ID from form data
    spotID := c.PostForm("spot_id")
    if spotID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id is required"})
        return
    }
    
    // Parse timestamp if provided, otherwise use current time
    timestampStr := c.PostForm("timestamp")
    var timestamp time.Time
    var err error
    if timestampStr != "" {
        // Try multiple timestamp formats
        formats := []string{
            time.RFC3339,
            "2006-01-02T15:04:05.999999",  // Python's isoformat()
            "2006-01-02T15:04:05",         // isoformat without microseconds
            "2006-01-02 15:04:05",
        }
        
        var parseError error
        for _, format := range formats {
            timestamp, parseError = time.Parse(format, timestampStr)
            if parseError == nil {
                break
            }
        }
        
        if parseError != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format. Use ISO 8601/RFC3339"})
            return
        }
    } else {
        timestamp = time.Now()
    }
    
    // Get the uploaded file
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }
    
    // Validate file is an image
    if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file must be an image"})
        return
    }
    
    // Open the file
    src, err := file.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
        return
    }
    defer src.Close()
    
    // Generate a unique filename
    ext := filepath.Ext(file.Filename)
    uniqueID := uuid.New().String()
    s3Key := fmt.Sprintf("snapshots/%s/%s%s", spotID, uniqueID, ext)
    
    // Upload to S3
    _, err = s3Client.PutObject(&s3.PutObjectInput{
        Bucket:      aws.String("treblesurf-images"),
        Key:         aws.String(s3Key),
        Body:        src,
        ContentType: aws.String(file.Header.Get("Content-Type")),
        Metadata: map[string]*string{
            "SpotId":    aws.String(spotID),
            "Timestamp": aws.String(timestamp.Format(time.RFC3339)),
        },
    })
    
    if err != nil {
        log.Printf("S3 upload error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
        return
    }
    
    // Store metadata in DynamoDB
    snapshot := SpotSnapshot{
        SpotID:     spotID,
        ImageKey:   s3Key,
        Timestamp:  timestamp,
        UploadedAt: time.Now(),
    }
    
    item, err := dynamodbattribute.MarshalMap(snapshot)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process snapshot metadata"})
        return
    }
    
    // Use UpdateItem to ensure we're always storing the latest snapshot
    _, err = db.PutItem(&dynamodb.PutItemInput{
        TableName: aws.String("SpotSnapshots"),
        Item:      item,
    })
    
    if err != nil {
        fmt.Print("Failed to store snapshot:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store snapshot metadata"})
        return
    }
    
    
    c.JSON(http.StatusOK, gin.H{
        "message":   "Snapshot uploaded successfully",
        "image_key": s3Key,
    })
}

// GetLatestSnapshotHandler returns the latest snapshot for a specific spot
func GetLatestSnapshotHandler(c *gin.Context) {
    // This endpoint can be accessed by authenticated users
    spotID := c.Query("spot_id")
    if spotID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id parameter is required"})
        return
    }
    
    // Query DynamoDB for the latest snapshot
    result, err := db.GetItem(&dynamodb.GetItemInput{
        TableName: aws.String("SpotSnapshots"),
        Key: map[string]*dynamodb.AttributeValue{
            "spot_id": {S: aws.String(spotID)},
        },
    })
    
    if err != nil {
        fmt.Print("Failed to retrieve:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve snapshot data"})
        return
    }
    
    // Check if snapshot exists
    if result.Item == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "No snapshots available for this spot"})
        return
    }
    
    // Extract image key
    imageKey := ""
    timestampStr := ""
    
    if v, ok := result.Item["image_key"]; ok && v.S != nil {
        imageKey = *v.S
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid snapshot data"})
        return
    }
    
    if v, ok := result.Item["timestamp"]; ok && v.S != nil {
        timestampStr = *v.S
    }
    
    // Generate presigned URL for the image
    req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
        Bucket: aws.String("treblesurf-images"),
        Key:    aws.String(imageKey),
    })
    
    presignedURL, err := req.Presign(15 * time.Minute) // URL valid for 15 minutes
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL"})
        return
    }
    
    // Parse timestamp
    var timestamp time.Time
    if timestampStr != "" {
        timestamp, err = time.Parse(time.RFC3339, timestampStr)
        if err != nil {
            timestamp = time.Time{} // Use zero time if parsing fails
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "image_url":  presignedURL,
        "timestamp":  timestamp.Format(time.RFC3339),
        "image_key": imageKey,
    })
}