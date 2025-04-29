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

    r.POST("/auth/google", api.GoogleAuthHandler)
    r.GET("auth/validate", api.ValidateTokenHandler)
    
    r.GET("/regions", api.GetRegions)                      // Expects ?country=
    r.GET("/spots", api.GetSpots)                         // Expects ?region= &country=
    r.GET("/location", api.GetCoordinates)                // Expects ?country= &region= &spot=
    r.GET("/listSpotsForecast", api.GetListSpotsForecast) // Expects ?country= &region= &spots=
    r.GET("/regionForecast", api.GetRegionForecast)       // Expects ?country= &region=
    r.GET("/currentConditions", api.GetCurrentWeather)     // Expects ?country= &region= &spot=
    r.GET("/beforeAfterTide", api.GetBeforeAfterTides)    // Expects ?locationName=
    r.GET("/tideExtremes", api.GetDayTides)               // Expects ?locationName= &start=
    r.GET("/getLiveBuoyData", api.GetLiveBuoyData)
    r.GET("/getSingleBuoyData", api.GetSingleBuoyData)    // Expects ?buoyName=
    r.GET("/getLast24BuoyData", api.GetLast24HoursBuoyData) // Expects ?buoyName=
    r.GET("/getMultipleBuoyData", api.GetMultipleBuoyData)      // Expects ?buoys=
    r.GET("/buoyLocationInfo", api.BuoyLocationInfo)
    r.GET("/individualBuoyLocation", api.IndividualBuoyLocationInfo) // Expects ?buoyName=
    r.GET("/locationInfo", api.GetLocationInfo)   

    // Protected routes (auth required)
    authorized := r.Group("/")
    authorized.Use(api.AuthMiddleware())
    {
        authorized.GET("/forecast", api.GetSpotForecast)               // Expects ?country= &region= &spot=
        authorized.DELETE("/deleteMyAccount", api.DeleteMyAccount)
        authorized.POST("/submitSurfReport", api.SubmitCurrentSurfReport)
        authorized.GET("/getTodaySpotReports", api.RetrieveTodaysSurfReports) // Expects ?country= &region= &spot=
    }

    ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// Debug logging for API Gateway request

// Check if path starts with /api and strip it
if strings.HasPrefix(req.Path, "/api") {
    req.Path = strings.TrimPrefix(req.Path, "/api")
}

resp, err := ginLambda.ProxyWithContext(ctx, req)
    return resp, err
}

func main() {
	lambda.Start(Handler)
}