// Package main provides the API Gateway Lambda handler for the Treble Surf API.
package main

import (
	"context"
	"strings"
	httphandler "treblesurf-backend/internal/api"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func initialize() error {
	container, err := httphandler.NewContainer()
	if err != nil {
		return err
	}
	r := httphandler.SetupRouter(container)
	ginLambda = ginadapter.New(r)
	return nil
}

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Debug logging for API Gateway request

	// Check if path starts with /api and strip it
	req.Path = strings.TrimPrefix(req.Path, "/api")

	resp, err := ginLambda.ProxyWithContext(ctx, req)
	return resp, err
}

func main() {
	if err := initialize(); err != nil {
		panic(err)
	}
	lambda.Start(Handler)
}
