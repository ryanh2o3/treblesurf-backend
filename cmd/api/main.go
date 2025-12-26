package main

import (
	"context"
	"strings"
	"treblesurf-backend/internal/api"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
	container, err := api.NewContainer()
	if err != nil {
		panic(err)
	}
    r := api.SetupRouter(container)
    ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Debug logging for API Gateway request

	// Check if path starts with /api and strip it
	req.Path = strings.TrimPrefix(req.Path, "/api")

	resp, err := ginLambda.ProxyWithContext(ctx, req)
	return resp, err
}

func main() {
	lambda.Start(Handler)
}