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

    // Add logging middleware
    r.Use(func(c *gin.Context) {
        log.Printf("Method: %s", c.Request.Method)
        log.Printf("Full request URL: %s", c.Request.URL.String())
        c.Next()
    })

    // Strip /api prefix
    r.Use(func(c *gin.Context) {
        path := c.Request.URL.Path
        if strings.HasPrefix(path, "/api") {
            c.Request.URL.Path = strings.TrimPrefix(path, "/api")
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
	// Debug logging for API Gateway request
    log.Printf("API Gateway Path: %s", req.Path)
    log.Printf("API Gateway PathParameters: %v", req.PathParameters)
    log.Printf("API Gateway QueryStringParameters: %v", req.QueryStringParameters)
    log.Printf("API Gateway RequestContext: %+v", req.RequestContext)
    log.Printf("API Gateway Resource: %s", req.Resource)

    // Add proxy+ parameter handling
    if proxyPath, ok := req.PathParameters["proxy"]; ok {
        req.Path = "/" + proxyPath
        log.Printf("Updated path from proxy parameter: %s", req.Path)
    }

    return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}