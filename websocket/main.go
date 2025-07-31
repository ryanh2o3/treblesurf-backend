package main

import (
	"log"
	"treblesurf-backend/api" // Same package your HTTP handler uses

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func init() {
    // Initialize shared services
    if err := api.InitSessionService(); err != nil {
        log.Printf("Failed to initialize session service: %v", err)
    }
    // Init any other required services
}

func Handler(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    return api.HandleWebSocketEvent(req)
}

func main() {
    lambda.Start(Handler)
}