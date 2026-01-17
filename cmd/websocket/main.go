// Package main provides the WebSocket Lambda handler for the Treble Surf WebSocket API.
package main

import (
	"log"
	"treblesurf-backend/internal/app"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/websocket"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var websocketHandler *websocket.Handler

func initialize() error {
	cfg := config.MustLoad()

	application, err := app.NewWebSocket(cfg)
	if err != nil {
		return err
	}

	websocketHandler = application.Handler()
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
