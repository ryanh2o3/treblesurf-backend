// Package main provides the API Gateway Lambda handler for the Treble Surf API.
package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"treblesurf-backend/internal/app"
	"treblesurf-backend/internal/config"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func initialize() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	application, err := app.New(cfg)
	if err != nil {
		return err
	}
	ginLambda = ginadapter.New(application.GinEngine())
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
		slog.Error("failed to initialize api handler", slog.Any("error", err))
		os.Exit(1)
	}
	lambda.Start(Handler)
}
