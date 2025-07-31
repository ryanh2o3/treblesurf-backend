package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/golang-jwt/jwt"
)

// ConnectionInfo represents a WebSocket connection with user info
type ConnectionInfo struct {
    ConnectionID string    `json:"connection_id"`
    UserID       string    `json:"user_id"` // Email
    ConnectedAt  time.Time `json:"connected_at"`
    LastActive   time.Time `json:"last_active"`
    UserAgent    string    `json:"user_agent"`
    IPAddress    string    `json:"ip_address"`
}

// WebSocketMessage represents the structure of incoming messages
type WebSocketMessage struct {
    Action string          `json:"action"`
    Data   json.RawMessage `json:"data"`
}

// HandleWebSocketEvent handles WebSocket events from API Gateway
func HandleWebSocketEvent(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    log.Printf("WebSocket event: %s, ConnectionID: %s", req.RequestContext.RouteKey, req.RequestContext.ConnectionID)

    switch req.RequestContext.RouteKey {
    case "$connect":
        return handleConnect(req)
    case "$disconnect":
        return handleDisconnect(req)
    case "$default":
        return handleDefault(req)
    default:
        return handleCustomRoute(req)
    }
}


func handleConnect(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
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
    wsToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })

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
    connection := ConnectionInfo{
        ConnectionID: connectionID,
        UserID:       userID,
        ConnectedAt:  time.Now(),
        LastActive:   time.Now(),
        UserAgent:    req.Headers["User-Agent"],
        IPAddress:    getSourceIP(req),
    }

    if err := saveConnection(connection); err != nil {
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

func handleDisconnect(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    connectionID := req.RequestContext.ConnectionID

    // Delete the connection record
    if err := deleteConnection(connectionID); err != nil {
        log.Printf("Error deleting connection: %v", err)
    }

    log.Printf("Client disconnected: %s", connectionID)
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
        Body:       "Disconnected",
    }, nil
}

func handleDefault(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    var message WebSocketMessage
    if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusBadRequest,
            Body:       "Invalid message format",
        }, nil
    }

    // Update last active timestamp
    updateConnectionLastActive(req.RequestContext.ConnectionID)

    // Process based on the action
    if message.Action == "" {
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusBadRequest,
            Body:       "Missing action field",
        }, nil
    }

    // Here we'd handle different message types
    return handleCustomRoute(req)
}

func handleCustomRoute(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    var message WebSocketMessage
    if err := json.Unmarshal([]byte(req.Body), &message); err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusBadRequest,
            Body:       "Invalid message format",
        }, nil
    }

    switch message.Action {
    case "ping":
        return handlePing(req)
    case "subscribe":
        return handleSubscribe(req, message.Data)
    // Add more custom actions as needed
    default:
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusBadRequest,
            Body:       "Unknown action",
        }, nil
    }
}

func handlePing(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Send pong response
    response := map[string]interface{}{
        "action": "pong",
        "time":   time.Now().Format(time.RFC3339),
    }
    
    responseJSON, _ := json.Marshal(response)
    err := sendToConnection(req.RequestContext.ConnectionID, string(responseJSON), 
        req.RequestContext.DomainName, req.RequestContext.Stage)
    
    if err != nil {
        log.Printf("Error sending pong: %v", err)
    }
    
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
    }, nil
}

func handleSubscribe(req events.APIGatewayWebsocketProxyRequest, data json.RawMessage) (events.APIGatewayProxyResponse, error) {
    // Handle subscription logic
    // This is where you'd implement subscription to different channels/topics
    
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
    }, nil
}

// saveConnection stores connection info in DynamoDB
func saveConnection(conn ConnectionInfo) error {
    item, err := dynamodbattribute.MarshalMap(conn)
    if err != nil {
        return err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String("WebSocketConnections"),
        Item:      item,
    }

    _, err = db.PutItem(input)
    return err
}

// deleteConnection removes a connection from DynamoDB
func deleteConnection(connectionID string) error {
    input := &dynamodb.DeleteItemInput{
        TableName: aws.String("WebSocketConnections"),
        Key: map[string]*dynamodb.AttributeValue{
            "ConnectionID": {
                S: aws.String(connectionID),
            },
        },
    }

    _, err := db.DeleteItem(input)
    return err
}

// updateConnectionLastActive updates the LastActive timestamp for a connection
func updateConnectionLastActive(connectionID string) error {
    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("WebSocketConnections"),
        Key: map[string]*dynamodb.AttributeValue{
            "ConnectionID": {
                S: aws.String(connectionID),
            },
        },
        UpdateExpression: aws.String("set LastActive = :t"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":t": {
                S: aws.String(time.Now().Format(time.RFC3339)),
            },
        },
    }

    _, err := db.UpdateItem(input)
    return err
}

// sendToConnection sends a message to a specific WebSocket client
func sendToConnection(connectionID, message, endpoint, stage string) error {
    sess := session.Must(session.NewSession())
    
    // Create API Gateway Management API client
    apiEndpoint := fmt.Sprintf("https://%s/%s", endpoint, stage)
    client := apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(apiEndpoint))
    
    _, err := client.PostToConnection(&apigatewaymanagementapi.PostToConnectionInput{
        ConnectionId: aws.String(connectionID),
        Data:         []byte(message),
    })
    
    return err
}

// getSourceIP extracts the source IP from the request
func getSourceIP(req events.APIGatewayWebsocketProxyRequest) string {
    // Try CloudFront-specific header first
    if ip, ok := req.Headers["CloudFront-Viewer-Address"]; ok && ip != "" {
        return strings.Split(ip, ":")[0]
    }
    
    // Try standard headers
    if ip, ok := req.Headers["X-Forwarded-For"]; ok && ip != "" {
        // X-Forwarded-For may contain multiple IPs, take the first one
        ips := strings.Split(ip, ",")
        return strings.TrimSpace(ips[0])
    }
    
    // Fallback to source IP from API Gateway
    return req.RequestContext.Identity.SourceIP
}

// Add this function to your websocketHandler.go

// BroadcastToUsers sends a message to all connections for specific users
func BroadcastToUsers(userIDs []string, message interface{}, stage string) error {
    // Convert message to JSON
    msgJSON, err := json.Marshal(message)
    if err != nil {
        return err
    }
    
    // Get endpoint from environment variable
    endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
    if endpoint == "" {
        return fmt.Errorf("WEBSOCKET_API_ENDPOINT not configured")
    }
    
    // Query connections for these users
    connections, err := getConnectionsByUsers(userIDs)
    if err != nil {
        return err
    }
    
    // Send to each connection
    for _, conn := range connections {
        err := sendToConnection(conn.ConnectionID, string(msgJSON), endpoint, stage)
        if err != nil {
            log.Printf("Failed to send to connection %s: %v", conn.ConnectionID, err)
            // Consider removing stale connections here
        }
    }
    
    return nil
}

// getConnectionsByUsers queries DynamoDB for connections by user IDs
func getConnectionsByUsers(userIDs []string) ([]ConnectionInfo, error) {
    var connections []ConnectionInfo
    
    if len(userIDs) == 0 {
        return connections, nil
    }
    
    // Create filter condition for UserID IN [id1, id2, ...]
    // We need to build this with OR conditions since ValueList isn't available
    var filterCondition expression.ConditionBuilder
    
    // Start with the first user ID
    filterCondition = expression.Name("UserID").Equal(expression.Value(userIDs[0]))
    
    // Add OR conditions for each additional user ID
    for i := 1; i < len(userIDs); i++ {
        filterCondition = filterCondition.Or(expression.Name("UserID").Equal(expression.Value(userIDs[i])))
    }
    
    // Create expression with filter
    expr, err := expression.NewBuilder().WithFilter(filterCondition).Build()
    
    if err != nil {
        return nil, fmt.Errorf("error building expression: %v", err)
    }
    
    // Scan the connections table with filter
    input := &dynamodb.ScanInput{
        TableName:                 aws.String("WebSocketConnections"),
        FilterExpression:          expr.Filter(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
    }
    
    result, err := db.Scan(input)
    if err != nil {
        return nil, fmt.Errorf("error scanning connections: %v", err)
    }
    
    // Unmarshal the results
    err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &connections)
    if err != nil {
        return nil, fmt.Errorf("error unmarshaling connections: %v", err)
    }
    
    return connections, nil
}