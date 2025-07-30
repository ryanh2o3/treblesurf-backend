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

        // origin := c.Request.Header.Get("Origin")
        
        // // Allow specific origins (production and localhost for development)
        // allowedOrigins := []string{
        //     "https://treblesurf.com", 
        //     "http://localhost:5174", // Local development
        //     "http://localhost:5173",
        //     // Add any other local development URLs you need
        // }
        
        // // Check if origin is allowed
        // allowOrigin := ""
        // for _, allowed := range allowedOrigins {
        //     if origin == allowed {
        //         allowOrigin = origin
        //         break
        //     }
        // }
        
        // // If origin is allowed or running in development mode
        // if allowOrigin != "" {    
        //     c.Header("Access-Control-Allow-Origin", allowOrigin)
        //     c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        //     c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
        //     c.Header("Access-Control-Allow-Credentials", "true")
        // }
        
        // // Handle preflight OPTIONS requests
        // if c.Request.Method == "OPTIONS" {
        //     c.AbortWithStatus(204)
        //     return
        // })
        
        // Print all route patterns
        routes := r.Routes()
        log.Printf("=== Registered Routes ===")
        for _, route := range routes {
            log.Printf("Route Pattern: %s %s", route.Method, route.Path)
        }
        
        c.Next()
    })

    // Move the /api stripping middleware after the logging
    r.Use(func(c *gin.Context) {
        path := c.Request.URL.Path
        if strings.HasPrefix(path, "/api") {
            newPath := strings.TrimPrefix(path, "/api")
            c.Request.URL.Path = newPath
        }
        c.Next()
    })

    if err := api.InitSessionService(); err != nil {
        log.Printf("Failed to initialize session service: %v", err)
    }

    r.POST("/auth/google", api.GoogleAuthHandler)
    r.GET("auth/validate", api.ValidateTokenHandler)
    r.POST("/auth/logout", api.LogoutHandler)

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
    r.GET("/forecast", api.GetSpotForecast)               // Expects ?country= &region= &spot=


    // Protected routes (auth required)
    authorized := r.Group("/")
    // authorized.Use(api.ClientTypeMiddleware()) // First detect client type
    // authorized.Use(api.AdaptiveAuthMiddleware())
    authorized.Use(api.WebAuthMiddleware())
    {
        webModifyGroup := authorized.Group("/")
        // webModifyGroup.Use(func(c *gin.Context) {
        // // Skip this middleware for app clients
        // isAppClient, _ := c.Get("isAppClient")
        // if isAppClient.(bool) {
        //     c.Next()
        //     return
        // }
        // // Apply CSRF only for web clients
        // api.CSRFMiddleware()(c)
        // })
        webModifyGroup.Use(api.CSRFMiddleware())
        {
            webModifyGroup.POST("/submitSurfReport", api.SubmitCurrentSurfReport)
            webModifyGroup.DELETE("/deleteMyAccount", api.DeleteMyAccount)
            webModifyGroup.PUT("/setTheme", api.SetUserTheme)
            webModifyGroup.DELETE("/sessions/:sessionId", api.TerminateSessionHandler)

        // Other state-changing endpoints
        }
        authorized.GET("/sessions", api.GetUserSessionsHandler)
        authorized.GET("/getTheme", api.GetUserTheme)
        authorized.GET("/getTodaySpotReports", api.RetrieveTodaysSurfReports) // Expects ?country= &region= &spot=
        authorized.GET("getReportImage", api.GetReportImage) // Expects ?key=
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