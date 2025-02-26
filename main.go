package main

import (
	"context"
	"log"
	"strings"
	"treblesurf-backend/api"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
	r := gin.Default()

    // Enhanced logging middleware
    r.Use(func(c *gin.Context) {
        log.Printf("=== Request Debug ===")
        log.Printf("Method: %s", c.Request.Method)
        log.Printf("Full URL: %s", c.Request.URL.String())
        log.Printf("Path: %s", c.Request.URL.Path)
        log.Printf("Raw Query: %s", c.Request.URL.RawQuery)
        log.Printf("Parameters: %v", c.Params)
        log.Printf("Headers: %v", c.Request.Header)
        
        // Print all route patterns
        routes := r.Routes()
        log.Printf("=== Registered Routes ===")
        for _, route := range routes {
            log.Printf("Route Pattern: %s %s", route.Method, route.Path)
        }
        
        c.Next()
        
        // Log the response status
        log.Printf("=== Response ===")
        log.Printf("Status: %d", c.Writer.Status())
    })

    // Move the /api stripping middleware after the logging
    r.Use(func(c *gin.Context) {
        path := c.Request.URL.Path
        if strings.HasPrefix(path, "/api") {
            newPath := strings.TrimPrefix(path, "/api")
            log.Printf("Stripped /api prefix. Original: %s, New: %s", path, newPath)
            c.Request.URL.Path = newPath
        }
        c.Next()
    })
    
    r.GET("/regions", api.GetRegions)                      // Expects ?country=
    r.GET("/spots", api.GetSpots)                         // Expects ?region= &country=
    r.GET("/location", api.GetCoordinates)                // Expects ?country= &region= &spot=
    r.GET("/forecast", api.GetSpotForecast)               // Expects ?country= &region= &spot=
    r.GET("/listSpotsForecast", api.GetListSpotsForecast) // Expects ?country= &region= &spots=
    r.GET("/regionForecast", api.GetRegionForecast)       // Expects ?country= &region=
    //r.GET("/currentConditions", api.GetCurrentWeather)     // Expects ?country= &region= &spot=
    r.GET("/beforeAfterTide", api.GetBeforeAfterTides)    // Expects ?locationName=
    r.GET("/tideExtremes", api.GetDayTides)               // Expects ?locationName= &start=
    r.GET("/getLiveBuoyData", api.GetLiveBuoyData)
    r.GET("/getSingleBuoyData", api.GetSingleBuoyData)    // Expects ?buoyName=
    r.GET("/getLast24BuoyData", api.GetLast24HoursBuoyData) // Expects ?buoyName=
    r.POST("/submitSurfReport", api.SubmitCurrentSurfReport)
    r.GET("/getTodaySpotReports", api.RetrieveTodaysSurfReports) // Expects ?country= &region= &spot=
    r.GET("/getMultipleBuoyData", api.GetMultipleBuoyData)      // Expects ?buoys=
    r.GET("/buoyLocationInfo", api.BuoyLocationInfo)
    r.GET("/individualBuoyLocation", api.IndividualBuoyLocationInfo) // Expects ?buoyName=
    r.GET("/locationInfo", api.GetLocationInfo)   
    ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// Debug logging for API Gateway request
log.Printf("=== Lambda Handler Debug ===")
log.Printf("API Gateway Path: %s", req.Path)
log.Printf("API Gateway HTTP Method: %s", req.HTTPMethod)
log.Printf("API Gateway Query Parameters: %v", req.QueryStringParameters)
log.Printf("API Gateway Path Parameters: %v", req.PathParameters)

// Check if path starts with /api and strip it
if strings.HasPrefix(req.Path, "/api") {
    req.Path = strings.TrimPrefix(req.Path, "/api")
    log.Printf("Stripped /api prefix in Lambda Handler. New path: %s", req.Path)
}

resp, err := ginLambda.ProxyWithContext(ctx, req)
    log.Printf("Lambda Response Status: %d", resp.StatusCode)
    log.Printf("Lambda Response Body: %s", resp.Body)
    return resp, err
}

func main() {
	lambda.Start(Handler)
}