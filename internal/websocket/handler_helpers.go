package websocket

import (
	"net/http"
	"time"

	"treblesurf-backend/internal/model"

	"github.com/aws/aws-lambda-go/events"
)

// validateWebSocketToken validates the WebSocket token and extracts user information.
func (h *Handler) validateWebSocketToken(token string) (userID, sessionID string, err error) {
	if token == "" {
		return "", "", nil
	}

	wsToken, err := h.websocketService.ValidateWebSocketToken(token)
	if err != nil || !wsToken.Valid {
		return "", "", err
	}

	// Extract email from the new JWT-based WebSocket token
	email, err := h.websocketService.GetEmailFromToken(wsToken)
	if err != nil {
		return "", "", err
	}

	return email, "", nil
}

// createConnectionInfo creates a ConnectionInfo from the request.

//nolint:gocritic // AWS Lambda handler signature cannot be changed
func (h *Handler) createConnectionInfo(
	connectionID, userID string,
	req events.APIGatewayWebsocketProxyRequest,
) *model.ConnectionInfo {
	return &model.ConnectionInfo{
		ConnectionID: connectionID,
		UserID:       userID,
		ConnectedAt:  time.Unix(req.RequestContext.RequestTimeEpoch, 0),
		LastActive:   time.Unix(req.RequestContext.RequestTimeEpoch, 0),
		UserAgent:    req.Headers["User-Agent"],
		IPAddress:    h.websocketService.GetSourceIP(req.Headers),
	}
}

// unauthorizedResponse returns an unauthorized response.
func unauthorizedResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       "Authentication required",
	}
}

// invalidTokenResponse returns an invalid token response.
func invalidTokenResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       "Invalid token",
	}
}

// connectionFailedResponse returns a connection failed response.
func connectionFailedResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Failed to establish connection",
	}
}

// successResponse returns a success response.
func successResponse(message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       message,
	}
}
