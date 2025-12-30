// Package main provides the WebSocket Lambda handler for the Treble Surf WebSocket API.
package main

import (
	"fmt"
	"log"
	"os"
	"treblesurf-backend/internal/service"
	"treblesurf-backend/internal/storage"
	"treblesurf-backend/internal/websocket"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var websocketHandler *websocket.WebSocketHandler

func initialize() error {
	// Get configuration from environment
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1" // default
	}

	// Initialize storage
	dbStorage, err := storage.NewDynamoDBStorage(region)
	if err != nil {
		return fmt.Errorf("failed to initialize DynamoDB storage: %w", err)
	}

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// Initialize WebSocket service
	websocketService := service.NewWebSocketService(dbStorage, []byte(jwtSecret))

	// Initialize WebSocket handler
	websocketHandler = websocket.NewWebSocketHandler(websocketService)

	log.Println("WebSocket handler initialized successfully")
	return nil
}

//nolint:gocritic // AWS Lambda handler signature is fixed by AWS SDK
func Handler(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return websocketHandler.HandleWebSocketEvent(req)
}

func main() {
	if err := initialize(); err != nil {
		log.Fatal(err)
	}
	lambda.Start(Handler)
}