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

// Handler handles WebSocket events from API Gateway.
type Handler struct {
	websocketService *service.WebSocketService
}

// NewHandler creates a new WebSocket handler instance.
func NewHandler(websocketService *service.WebSocketService) *Handler {
	return &Handler{
		websocketService: websocketService,
	}
}

// HandleWebSocketEvent handles WebSocket events from API Gateway

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) HandleWebSocketEvent(
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

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleConnect(
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

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleDisconnect(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	if err := h.websocketService.DeleteConnection(connectionID); err != nil {
		log.Printf("Error deleting connection: %v", err)
	}

	log.Printf("Client disconnected: %s", connectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Disconnected",
	}, nil
}

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleDefault(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	var message model.WebSocketMessage
	if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid message format",
		}, nil
	}

	if err := h.websocketService.UpdateConnectionLastActive(req.RequestContext.ConnectionID); err != nil {
		log.Printf("Warning: Failed to update connection last active: %v", err)
	}

	if message.Action == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Missing action field",
		}, nil
	}

	return h.handleCustomRoute(req)
}

// handleCustomRoute processes custom WebSocket messages

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleCustomRoute(
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

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleSubscribeAction(
	req events.APIGatewayWebsocketProxyRequest, data json.RawMessage,
) (events.APIGatewayProxyResponse, error) {
	var subRequest model.SubscriptionRequest
	if err := json.Unmarshal(data, &subRequest); err != nil {
		log.Printf("Error parsing subscription data: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid subscription data",
		}, nil
	}

	connectionID := req.RequestContext.ConnectionID

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

	if saveErr := h.websocketService.SaveSubscription(spotIdentifier, userID, connectionID); saveErr != nil {
		log.Printf("Failed to save subscription: %v", saveErr)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to subscribe",
		}, nil
	}

	if updateErr := h.websocketService.UpdateConnectionSpot(connectionID, spotIdentifier); updateErr != nil {
		log.Printf("Warning: Failed to update connection spot: %v", updateErr)
	}

	response := h.websocketService.CreateSubscriptionResponse(spotIdentifier)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process subscription",
		}, nil
	}
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

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handlePingAction(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	if err := h.websocketService.UpdateConnectionLastActive(connectionID); err != nil {
		log.Printf("Warning: Failed to update connection last active: %v", err)
	}

	response := h.websocketService.CreatePongResponse()
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process ping",
		}, nil
	}
	if err := h.websocketService.SendToConnection(connectionID, string(responseJSON)); err != nil {
		log.Printf("Warning: Failed to send message to connection: %v", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Ping received",
	}, nil
}
