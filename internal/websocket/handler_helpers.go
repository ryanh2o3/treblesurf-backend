package websocket

import (
	"net/http"
	"time"

	"treblesurf-backend/internal/model"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt"
)

// validateWebSocketToken validates the WebSocket token and extracts user information.
func (h *WebSocketHandler) validateWebSocketToken(token string) (string, string, error) {
	if token == "" {
		return "", "", nil
	}

	wsToken, err := h.websocketService.ValidateWebSocketToken(token)
	if err != nil || !wsToken.Valid {
		return "", "", err
	}

	claims, ok := wsToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", nil
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", "", nil
	}

	sessionID, _ := claims["session_id"].(string)
	return userID, sessionID, nil
}

// createConnectionInfo creates a ConnectionInfo from the request.
func (h *WebSocketHandler) createConnectionInfo(connectionID, userID string, req events.APIGatewayWebsocketProxyRequest) *model.ConnectionInfo {
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

// invalidTokenFormatResponse returns an invalid token format response.
func invalidTokenFormatResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Invalid token format",
	}
}

// invalidTokenPayloadResponse returns an invalid token payload response.
func invalidTokenPayloadResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Body:       "Invalid token payload",
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

