package websocket

import (
	"net/http"
	"time"

	"treblesurf-backend/internal/model"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt"
)

// validateWebSocketToken validates the WebSocket token and extracts user information.
func (h *WebSocketHandler) validateWebSocketToken(token string) (userID, sessionID string, err error) {
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

	var ok2 bool
	userID, ok2 = claims["user_id"].(string)
	if !ok2 {
		return "", "", nil
	}

	if sessionIDVal, ok2 := claims["session_id"].(string); ok2 {
		sessionID = sessionIDVal
	}
	// Note: sessionID is optional, empty string is acceptable
	return userID, sessionID, nil
}

// createConnectionInfo creates a ConnectionInfo from the request.
//nolint:gocritic // AWS Lambda handler signature is fixed by AWS SDK
func (h *WebSocketHandler) createConnectionInfo(
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

