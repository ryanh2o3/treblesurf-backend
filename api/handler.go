package api

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gin-gonic/gin"
)

var db *dynamodb.DynamoDB

func init() {
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-west-1"),
    }))
    db = dynamodb.New(sess)
}

type Location struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type Forecast struct {
    DateForecastedFor string `json:"dateForecastedFor"`
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

    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    // Try for up to 48 hours in the past (adjust this number as needed)
    for i := 0; i < 48; i++ {
        searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
        dateStr := searchTime.Format("2006-01-02 15")
        

        forecast, err := queryForecastByDateTime(spotName, regionName, countryName, dateStr)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        
        if forecast != nil {
            c.JSON(http.StatusOK, forecast)
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func queryForecastByDateTime(spotName, regionName, countryName, dateTime string) ([]map[string]interface{}, error) {
    print(dateTime)
    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfSpotForecastData"),
        KeyConditionExpression: aws.String("forecastDate = :dateTime AND begins_with(country_region_spot, :location)"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":dateTime": {
                S: aws.String(dateTime),
            },
            ":location": {
                S: aws.String(fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)),
            },
        },
        ScanIndexForward: aws.Bool(true), 
    }

    result, err := db.Query(input)
    if err != nil {
        return nil, err
    }

    if len(result.Items) == 0 {
        return nil, nil
    }

    var forecasts []map[string]interface{}
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
    if err != nil {
        return nil, err
    }

    return forecasts, nil
}

func queryForecastByDateTimeLast(spotName, regionName, countryName, dateTime string) (map[string]interface{}, error) {
    print(dateTime)
    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfSpotForecastData"),
        KeyConditionExpression: aws.String("forecastDate = :dateTime AND begins_with(country_region_spot, :location)"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":dateTime": {
                S: aws.String(dateTime),
            },
            ":location": {
                S: aws.String(fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)),
            },
        },
        ScanIndexForward: aws.Bool(true), 
    }

    result, err := db.Query(input)
    if err != nil {
        return nil, err
    }

    if len(result.Items) == 0 {
        return nil, nil
    }

    var forecasts map[string]interface{}
    err = dynamodbattribute.UnmarshalMap(result.Items[0], &forecasts)
    if err != nil {
        return nil, err
    }

    return forecasts, nil
}


func GetListSpotsForecast(c *gin.Context) {
    spotsStr := c.Query("spots")
    spots := strings.Split(spotsStr, ",")
    regionName := c.Query("region")
    countryName := c.Query("country")
    print(spots)
    var allForecasts []map[string]interface{}

    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    for _, spotName := range spots {
        log.Printf("Getting forecast for %s", spotName)
        // Try for up to 48 hours in the past
        for i := 0; i < 48; i++ {
            searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
            dateStr := searchTime.Format("2006-01-02 15")

            forecast, err := queryForecastByDateTime(spotName, regionName, countryName, dateStr)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            
            if forecast != nil {
                // Filter forecasts to get current hour plus every 3rd hour
                var filteredForecasts []map[string]interface{}
                for i := 0; i < len(forecast); i++ {
                    if i == 0 || i%3 == 0 {  // Keep first hour and every 3rd hour after that
                        filteredForecasts = append(filteredForecasts, forecast[i])
                    }
                }
                allForecasts = append(allForecasts, filteredForecasts...)
                break
            }
        }
        
    }

    c.JSON(http.StatusOK, allForecasts)
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

    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    // Try for up to 48 hours in the past (adjust this number as needed)
    for i := 0; i < 48; i++ {
        searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
        dateStr := searchTime.Format("2006-01-02 15")
        

        forecast, err := queryForecastByDateTimeLast(spotName, regionName, countryName, dateStr)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        
        if forecast != nil {
            c.JSON(http.StatusOK, forecast)
            return
        }
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
        data := getBuoyData(buoy, "string", 1)
        buoyData = append(buoyData, data[1])
    }

    c.JSON(http.StatusOK, buoyData)
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
        data = getBuoyData(buoyName, dateStr, 1)[1]
        if data != nil {
            break
        }
    }

    c.JSON(http.StatusOK, data)
}

func GetLast24HoursBuoyData(c *gin.Context) {
    buoyName := c.Query("buoyName")
    var data []map[string]interface{}
    // Start from current time rounded down to the nearest hour
    now := time.Now()
    currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

    for i := 0; i < 12; i++ {
        searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
        dateStr := searchTime.UTC().Format("2006-01-02T15:00:00Z")
        data = getBuoyData(buoyName, dateStr, 24)
        if data != nil {
            break
        }
    }
    c.JSON(http.StatusOK, data)
}

func SubmitCurrentSurfReport(c *gin.Context) {
    var report map[string]string
    if err := c.BindJSON(&report); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if !isValidSwellSize(report["swellSize"]) || !isValidWindAmount(report["windAmount"]) || !isValidWindDirection(report["windDirection"]) || !isValidSurfConditions(report["surfConditions"]) || !isValidSurfDifficulty(report["surfDifficulty"]) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report data"})
        return
    }

    currentTime := time.Now()
    currentDate := currentTime.Format("2006-01-02")

    input := &dynamodb.PutItemInput{
        TableName: aws.String("SurfReports"),
        Item: map[string]*dynamodb.AttributeValue{
            "Country": {
                S: aws.String(report["country"]),
            },
            "Region": {
                S: aws.String(report["region"]),
            },
            "Spot": {
                S: aws.String(report["spot"]),
            },
            "Date": {
                S: aws.String(currentDate),
            },
            "SwellSize": {
                S: aws.String(report["swellSize"]),
            },
            "WindAmount": {
                S: aws.String(report["windAmount"]),
            },
            "WindDirection": {
                S: aws.String(report["windDirection"]),
            },
            "SurfConditions": {
                S: aws.String(report["surfConditions"]),
            },
            "SurfDifficulty": {
                S: aws.String(report["surfDifficulty"]),
            },
            "Time": {
                S: aws.String(currentTime.String()),
            },
        },
    }

    _, err := db.PutItem(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

func RetrieveTodaysSurfReports(c *gin.Context) {
    countryName := c.Query("country")
    regionName := c.Query("region")
    spotName := c.Query("spot")

    currentTime := time.Now()
    currentDate := currentTime.Format("2006-01-02")
    input := &dynamodb.QueryInput{
        TableName: aws.String("SurfReports"),
        KeyConditionExpression: aws.String("Country = :country AND Region = :region AND Spot = :spot AND Date = :date"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":country": {S: aws.String(countryName)},
            ":region":  {S: aws.String(regionName)},
            ":spot":    {S: aws.String(spotName)},
            ":date":    {S: aws.String(currentDate)},
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
    c.JSON(http.StatusOK, reports)
}

func GetMultipleBuoyData(c *gin.Context) {
    buoys := c.QueryArray("buoys")
    var values []map[string]interface{}

    for _, buoy := range buoys {
        data := getBuoyData(buoy, "string", 1)
        values = append(values, data[1])
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

func getBuoyData(buoyName string, dateStr string, limit int64) []map[string]interface{} {
    var buoyData []map[string]interface{}
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
        Limit:           aws.Int64(limit),     // We only want the most recent reading
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

    
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &buoyData)
    if err != nil {
        log.Printf("Error unmarshalling buoy data: %v", err)
        return nil
    }
    return buoyData
    
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
    validConditions := []string{"clean", "messy", "okay"}
    return contains(validConditions, surfConditions)
}

func isValidSurfDifficulty(surfDifficulty string) bool {
    validDifficulties := []string{"lulls", "consistent", "relentless"}
    return contains(validDifficulties, surfDifficulty)
}
