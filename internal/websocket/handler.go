// Package websocket provides WebSocket handlers for API Gateway WebSocket connections.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	slog.Info("websocket event",
		slog.String("route", req.RequestContext.RouteKey),
		slog.String("connection_id", req.RequestContext.ConnectionID),
	)

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
	ctx := context.Background()

	userID, sessionID, err := h.validateWebSocketToken(token)
	if err != nil || userID == "" {
		if token == "" {
			return unauthorizedResponse(), nil
		}
		slog.Warn("invalid WebSocket token", slog.Any("error", err))
		return invalidTokenResponse(), nil
	}

	connection := h.createConnectionInfo(connectionID, userID, req)
	if err := h.websocketService.SaveConnection(ctx, connection); err != nil {
		slog.Warn("failed to save connection", slog.Any("error", err))
		return connectionFailedResponse(), nil
	}

	slog.Info("client connected",
		slog.String("connection_id", connectionID),
		slog.String("user_id", userID),
		slog.String("session_id", sessionID),
	)
	return successResponse("Connected"), nil
}

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) handleDisconnect(
	req events.APIGatewayWebsocketProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID
	ctx := context.Background()

	if err := h.websocketService.DeleteConnection(ctx, connectionID); err != nil {
		slog.Warn("error deleting connection", slog.Any("error", err))
	}

	slog.Info("client disconnected", slog.String("connection_id", connectionID))
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

	if err := h.websocketService.UpdateConnectionLastActive(context.Background(), req.RequestContext.ConnectionID); err != nil {
		slog.Warn("failed to update connection last active", slog.Any("error", err))
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
		slog.Warn("error parsing message", slog.Any("error", err))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid message format",
		}, nil
	}

	connectionID := req.RequestContext.ConnectionID
	slog.Info("received action", slog.String("action", message.Action), slog.String("connection_id", connectionID))

	switch message.Action {
	case "subscribe":
		return h.handleSubscribeAction(req, message.Data)
	case "ping":
		return h.handlePingAction(req)
	default:
		slog.Warn("unknown action", slog.String("action", message.Action))
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
		slog.Warn("error parsing subscription data", slog.Any("error", err))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid subscription data",
		}, nil
	}

	connectionID := req.RequestContext.ConnectionID

	connection, err := h.websocketService.GetConnection(context.Background(), connectionID)
	if err != nil {
		slog.Warn("error getting connection info", slog.Any("error", err))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process subscription",
		}, nil
	}

	userID := connection.UserID
	spotIdentifier := fmt.Sprintf("%s/%s/%s", subRequest.Country, subRequest.Region, subRequest.Spot)

	if saveErr := h.websocketService.SaveSubscription(context.Background(), spotIdentifier, userID, connectionID); saveErr != nil {
		slog.Warn("failed to save subscription", slog.Any("error", saveErr))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to subscribe",
		}, nil
	}

	if updateErr := h.websocketService.UpdateConnectionSpot(context.Background(), connectionID, spotIdentifier); updateErr != nil {
		slog.Warn("failed to update connection spot", slog.Any("error", updateErr))
	}

	response := h.websocketService.CreateSubscriptionResponse(spotIdentifier)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		slog.Warn("failed to marshal response", slog.Any("error", err))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process subscription",
		}, nil
	}
	if err := h.websocketService.SendToConnection(connectionID, string(responseJSON)); err != nil {
		slog.Warn("failed to send message to connection", slog.Any("error", err))
	}

	slog.Info("user subscribed to spot",
		slog.String("user_id", userID),
		slog.String("spot", spotIdentifier),
		slog.String("connection_id", connectionID),
	)
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

	if err := h.websocketService.UpdateConnectionLastActive(context.Background(), connectionID); err != nil {
		slog.Warn("failed to update connection last active", slog.Any("error", err))
	}

	response := h.websocketService.CreatePongResponse()
	responseJSON, err := json.Marshal(response)
	if err != nil {
		slog.Warn("failed to marshal response", slog.Any("error", err))
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process ping",
		}, nil
	}
	if err := h.websocketService.SendToConnection(connectionID, string(responseJSON)); err != nil {
		slog.Warn("failed to send message to connection", slog.Any("error", err))
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Ping received",
	}, nil
}
