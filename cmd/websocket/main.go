package main

import (
	"log"
	"os"
	"treblesurf-backend/internal/service"
	"treblesurf-backend/internal/storage"
	"treblesurf-backend/internal/websocket"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var websocketHandler *websocket.WebSocketHandler

func init() {
	// Get configuration from environment
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1" // default
	}

	// Initialize storage
	dbStorage, err := storage.NewDynamoDBStorage(region)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB storage: %v", err)
	}

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Initialize WebSocket service
	websocketService := service.NewWebSocketService(dbStorage, []byte(jwtSecret))

	// Initialize WebSocket handler
	websocketHandler = websocket.NewWebSocketHandler(websocketService)

	log.Println("WebSocket handler initialized successfully")
}

func Handler(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return websocketHandler.HandleWebSocketEvent(req)
}

func main() {
	lambda.Start(Handler)
}