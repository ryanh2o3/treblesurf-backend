package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt"
)

type WebSocketHandler struct {
	websocketService *service.WebSocketService
}

func NewWebSocketHandler(websocketService *service.WebSocketService) *WebSocketHandler {
	return &WebSocketHandler{
		websocketService: websocketService,
	}
}

// HandleWebSocketEvent handles WebSocket events from API Gateway
func (h *WebSocketHandler) HandleWebSocketEvent(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
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

func (h *WebSocketHandler) handleConnect(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	// Parse token from query parameters
	token := req.QueryStringParameters["token"]
	if token == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Authentication required",
		}, nil
	}

	// Parse the temporary token
	wsToken, err := h.websocketService.ValidateWebSocketToken(token)
	if err != nil || !wsToken.Valid {
		log.Printf("Invalid WebSocket token: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Invalid token",
		}, nil
	}

	// Extract claims
	claims, ok := wsToken.Claims.(jwt.MapClaims)
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Invalid token format",
		}, nil
	}

	// Get user ID from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid token payload",
		}, nil
	}

	// Get session ID for reference (optional)
	sessionID, _ := claims["session_id"].(string)

	// Store connection info in DynamoDB
	connection := &model.ConnectionInfo{
		ConnectionID: connectionID,
		UserID:       userID,
		ConnectedAt:  time.Unix(req.RequestContext.RequestTimeEpoch, 0),
		LastActive:   time.Unix(req.RequestContext.RequestTimeEpoch, 0),
		UserAgent:    req.Headers["User-Agent"],
		IPAddress:    h.websocketService.GetSourceIP(req.Headers),
	}

	if err := h.websocketService.SaveConnection(connection); err != nil {
		log.Printf("Failed to save connection: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to establish connection",
		}, nil
	}

	log.Printf("Client connected: %s, User: %s, Session: %s", connectionID, userID, sessionID)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Connected",
	}, nil
}

func (h *WebSocketHandler) handleDisconnect(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
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

func (h *WebSocketHandler) handleDefault(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	var message model.WebSocketMessage
	if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid message format",
		}, nil
	}

	// Update last active timestamp
	h.websocketService.UpdateConnectionLastActive(req.RequestContext.ConnectionID)

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
func (h *WebSocketHandler) handleCustomRoute(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
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
func (h *WebSocketHandler) handleSubscribeAction(req events.APIGatewayWebsocketProxyRequest, data json.RawMessage) (events.APIGatewayProxyResponse, error) {
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
	h.websocketService.UpdateConnectionSpot(connectionID, spotIdentifier)

	// Send confirmation back to client
	response := h.websocketService.CreateSubscriptionResponse(spotIdentifier)
	responseJSON, _ := json.Marshal(response)
	h.websocketService.SendToConnection(connectionID, string(responseJSON))

	log.Printf("User %s subscribed to spot: %s via connection %s", userID, spotIdentifier, connectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Subscribed",
	}, nil
}

// handlePingAction responds to ping messages to keep the connection alive
func (h *WebSocketHandler) handlePingAction(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	// Update last active time
	h.websocketService.UpdateConnectionLastActive(connectionID)

	// Send pong response
	response := h.websocketService.CreatePongResponse()
	responseJSON, _ := json.Marshal(response)
	h.websocketService.SendToConnection(connectionID, string(responseJSON))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Ping received",
	}, nil
}
