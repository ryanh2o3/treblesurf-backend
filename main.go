package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func main() {
	lambda.Start(Handler)
}

func init() {
	r := gin.Default()
    r.GET("/regions/:countryName", getRegions)
    r.GET("/spots/:countryName/:regionName", getSpots)
    r.GET("/coordinates/:countryName/:regionName/:spotName", getCoordinates)
    r.GET("/spotForecast/:countryName/:regionName/:spotName", getSpotForecast)
    r.GET("/listSpotsForecast/:countryName/:regionName", getListSpotsForecast)
    r.GET("/regionForecast/:countryName/:regionName", getRegionForecast)
    r.GET("/currentWeather/:countryName/:regionName/:spotName", getCurrentWeather)
    r.GET("/beforeAfterTides/:locationName", getBeforeAfterTides)
    r.GET("/dayTides/:locationName/:startDay", getDayTides)
    r.GET("/liveBuoyData", getLiveBuoyData)
    r.GET("/singleBuoyData/:buoyName", getSingleBuoyData)
    r.GET("/last24HoursBuoyData/:buoyName", getLast24HoursBuoyData)
    r.POST("/submitCurrentSurfReport", submitCurrentSurfReport)
    r.GET("/todaysSurfReports/:countryName/:regionName/:spotName", retrieveTodaysSurfReports)
    r.GET("/multipleBuoyData", getMultipleBuoyData)
    r.GET("/buoyLocationInfo", buoyLocationInfo)
    r.GET("/individualBuoyLocationInfo/:regionName/:buoyName", individualBuoyLocationInfo)
    r.GET("/locationInfo/:countryName/:regionName/:spotName", getLocationInfo)
    ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}
