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
    
    r.GET("/regions/:countryName", api.GetRegions)
    r.GET("/spots/:countryName/:regionName", api.GetSpots)
    r.GET("/coordinates/:countryName/:regionName/:spotName", api.GetCoordinates)
    r.GET("/spotForecast/:countryName/:regionName/:spotName", api.GetSpotForecast)
    r.GET("/listSpotsForecast/:countryName/:regionName", api.GetListSpotsForecast)
    r.GET("/regionForecast/:countryName/:regionName", api.GetRegionForecast)
    r.GET("/currentWeather/:countryName/:regionName/:spotName", api.GetCurrentWeather)
    r.GET("/beforeAfterTides/:locationName", api.GetBeforeAfterTides)
    r.GET("/dayTides/:locationName/:startDay", api.GetDayTides)
    r.GET("/liveBuoyData", api.GetLiveBuoyData)
    r.GET("/singleBuoyData/:buoyName", api.GetSingleBuoyData)
    r.GET("/last24HoursBuoyData/:buoyName", api.GetLast24HoursBuoyData)
    r.POST("/submitCurrentSurfReport", api.SubmitCurrentSurfReport)
    r.GET("/todaysSurfReports/:countryName/:regionName/:spotName", api.RetrieveTodaysSurfReports)
    r.GET("/multipleBuoyData", api.GetMultipleBuoyData)
    r.GET("/buoyLocationInfo", api.BuoyLocationInfo)
    r.GET("/individualBuoyLocationInfo/:regionName/:buoyName", api.IndividualBuoyLocationInfo)
    r.GET("/locationInfo/:countryName/:regionName/:spotName", api.GetLocationInfo)
    ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

    return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}