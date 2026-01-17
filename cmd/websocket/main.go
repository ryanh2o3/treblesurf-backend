// Package main provides the WebSocket Lambda handler for the Treble Surf WebSocket API.
package main

import (
	"fmt"
	"log"
	"os"
	repodynamo "treblesurf-backend/internal/repository/dynamodb"
	"treblesurf-backend/internal/service"
	"treblesurf-backend/internal/storage"
	"treblesurf-backend/internal/websocket"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var websocketHandler *websocket.Handler

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
	dynamoClient := dbStorage.GetDynamoDBClient()

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// Initialize WebSocket repositories/service
	websocketRepo := repodynamo.NewWebSocketRepo(dynamoClient, "WebSocketConnections")
	subscriptionRepo := repodynamo.NewSpotSubscriptionRepo(dynamoClient, "SpotSubscriptions")
	websocketService := service.NewWebSocketService(websocketRepo, subscriptionRepo, []byte(jwtSecret))

	// Initialize WebSocket handler
	websocketHandler = websocket.NewHandler(websocketService)

	log.Println("WebSocket handler initialized successfully")
	return nil
}

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func Handler(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return websocketHandler.HandleWebSocketEvent(req)
}

func main() {
	if err := initialize(); err != nil {
		log.Fatal(err)
	}
	lambda.Start(Handler)
}
