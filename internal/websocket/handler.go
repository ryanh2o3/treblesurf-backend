// Package websocket provides WebSocket handlers for API Gateway WebSocket connections.
package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/aws/aws-lambda-go/events"
)

// WebSocketHandler handles WebSocket events from API Gateway.
//nolint:revive // Name stuttering is acceptable for clarity
type WebSocketHandler struct {
	websocketService *service.WebSocketService
}

// NewWebSocketHandler creates a new WebSocket handler instance.
func NewWebSocketHandler(websocketService *service.WebSocketService) *WebSocketHandler {
	return &WebSocketHandler{
		websocketService: websocketService,
	}
}

// HandleWebSocketEvent handles WebSocket events from API Gateway
func (h *WebSocketHandler) HandleWebSocketEvent(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket event: %s, ConnectionID: %s", req.RequestContext.RouteKey, req.RequestContext.ConnectionID)

	switch req.RequestContext.RouteKey {
	case "$connect":
		return h.handleConnect(req)
	case "$disconnect":
		return h.handleDisconnect(req)
	case "$default":
		return h.handleDefault(req)
	default:
		return h.handleCustomRoute(req)
	}
}

func (h *WebSocketHandler) handleConnect(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID
	token := req.QueryStringParameters["token"]

	userID, sessionID, err := h.validateWebSocketToken(token)
	if err != nil || userID == "" {
		if token == "" {
			return unauthorizedResponse(), nil
		}
		log.Printf("Invalid WebSocket token: %v", err)
		return invalidTokenResponse(), nil
	}

	connection := h.createConnectionInfo(connectionID, userID, req)
	if err := h.websocketService.SaveConnection(connection); err != nil {
		log.Printf("Failed to save connection: %v", err)
		return connectionFailedResponse(), nil
	}

	log.Printf("Client connected: %s, User: %s, Session: %s", connectionID, userID, sessionID)
	return successResponse("Connected"), nil
}

func (h *WebSocketHandler) handleDisconnect(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	// Delete the connection record
	if err := h.websocketService.DeleteConnection(connectionID); err != nil {
		log.Printf("Error deleting connection: %v", err)
	}

	log.Printf("Client disconnected: %s", connectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Disconnected",
	}, nil
}

func (h *WebSocketHandler) handleDefault(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	var message model.WebSocketMessage
	if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid message format",
		}, nil
	}

	// Update last active timestamp
	if err := h.websocketService.UpdateConnectionLastActive(req.RequestContext.ConnectionID); err != nil {
		log.Printf("Warning: Failed to update connection last active: %v", err)
	}

	// Process based on the action
	if message.Action == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Missing action field",
		}, nil
	}

	// Here we'd handle different message types
	return h.handleCustomRoute(req)
}

// handleCustomRoute processes custom WebSocket messages
func (h *WebSocketHandler) handleCustomRoute(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	var message model.WebSocketMessage
	if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
		log.Printf("Error parsing message: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid message format",
		}, nil
	}

	connectionID := req.RequestContext.ConnectionID
	log.Printf("Received action: %s from connection: %s", message.Action, connectionID)

	switch message.Action {
	case "subscribe":
		return h.handleSubscribeAction(req, message.Data)
	case "ping":
		return h.handlePingAction(req)
	default:
		log.Printf("Unknown action: %s", message.Action)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       "Unknown action",
		}, nil
	}
}

// handleSubscribeAction processes WebSocket subscription requests
func (h *WebSocketHandler) handleSubscribeAction(
	req events.APIGatewayWebsocketProxyRequest, data json.RawMessage,
) (events.APIGatewayProxyResponse, error) {
	// Parse the subscription data
	var subRequest model.SubscriptionRequest
	if err := json.Unmarshal(data, &subRequest); err != nil {
		log.Printf("Error parsing subscription data: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid subscription data",
		}, nil
	}

	connectionID := req.RequestContext.ConnectionID

	// Get the connection info to know which user is subscribing
	connection, err := h.websocketService.GetConnection(connectionID)
	if err != nil {
		log.Printf("Error getting connection info: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process subscription",
		}, nil
	}

	userID := connection.UserID
	spotIdentifier := fmt.Sprintf("%s/%s/%s", subRequest.Country, subRequest.Region, subRequest.Spot)

	// Store the subscription in DynamoDB
	if err := h.websocketService.SaveSubscription(spotIdentifier, userID, connectionID); err != nil {
		log.Printf("Failed to save subscription: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to subscribe",
		}, nil
	}

	// Update connection metadata with current spot
	if err := h.websocketService.UpdateConnectionSpot(connectionID, spotIdentifier); err != nil {
		log.Printf("Warning: Failed to update connection spot: %v", err)
	}

	// Send confirmation back to client
	response := h.websocketService.CreateSubscriptionResponse(spotIdentifier)
	responseJSON, _ := json.Marshal(response)
	if err := h.websocketService.SendToConnection(connectionID, string(responseJSON)); err != nil {
		log.Printf("Warning: Failed to send message to connection: %v", err)
	}

	log.Printf("User %s subscribed to spot: %s via connection %s", userID, spotIdentifier, connectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Subscribed",
	}, nil
}

// handlePingAction responds to ping messages to keep the connection alive
func (h *WebSocketHandler) handlePingAction(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	// Update last active time
	if err := h.websocketService.UpdateConnectionLastActive(connectionID); err != nil {
		log.Printf("Warning: Failed to update connection last active: %v", err)
	}

	// Send pong response
	response := h.websocketService.CreatePongResponse()
	responseJSON, _ := json.Marshal(response)
	if err := h.websocketService.SendToConnection(connectionID, string(responseJSON)); err != nil {
		log.Printf("Warning: Failed to send message to connection: %v", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Ping received",
	}, nil
}
