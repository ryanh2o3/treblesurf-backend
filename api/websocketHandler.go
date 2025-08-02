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
	CurrentSpot  string    `json:"current_spot,omitempty"`
	TTL 		 int64 	   `json:"ttl"`
	
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

// handleCustomRoute processes custom WebSocket messages
func handleCustomRoute(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    var message WebSocketMessage
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
        return handleSubscribeAction(req, message.Data)
    case "ping":
        return handlePingAction(req)
    default:
        log.Printf("Unknown action: %s", message.Action)
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusOK,
            Body:       "Unknown action",
        }, nil
    }
}

// saveConnection stores connection info in DynamoDB
func saveConnection(conn ConnectionInfo) error {

	conn.TTL = time.Now().Add(24 * time.Hour).Unix()

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
            "connection_id": {
                S: aws.String(connectionID),
            },
        },
    }

    _, err := db.DeleteItem(input)
    return err
}

// updateConnectionLastActive updates the LastActive timestamp for a connection
func updateConnectionLastActive(connectionID string) error {
	newTTL := time.Now().Add(24 * time.Hour).Unix()

    input := &dynamodb.UpdateItemInput{
        TableName: aws.String("WebSocketConnections"),
        Key: map[string]*dynamodb.AttributeValue{
            "connection_id": {
                S: aws.String(connectionID),
            },
        },
        UpdateExpression: aws.String("set LastActive = :t"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":t": {
                S: aws.String(time.Now().Format(time.RFC3339)),
            },
			":ttl": {
                N: aws.String(fmt.Sprintf("%d", newTTL)),
            },
        },
    }

    _, err := db.UpdateItem(input)
    return err
}

// sendToConnection sends a message to a specific WebSocket client
func sendToConnection(connectionID, message string) error {
    // Get endpoint and stage from environment variables
    endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
    if endpoint == "" {
        return fmt.Errorf("WEBSOCKET_API_ENDPOINT not configured")
    }
    
    stage := os.Getenv("WEBSOCKET_API_STAGE") 
    if stage == "" {
        stage = "production" // Default stage name
    }
    
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
func BroadcastToUsers(userIDs []string, message interface{}) error {
    // Convert message to JSON
    msgJSON, err := json.Marshal(message)
    if err != nil {
		log.Print("json err", err)
        return err
    }
    
    // Query connections for these users
    connections, err := getConnectionsByUsers(userIDs)
    if err != nil {
		log.Print("conn err", err)
        return err
    }
	log.Print("connections", connections)
    
    // Send to each connection
    var sendErr error
    for _, conn := range connections {
        err := sendToConnection(conn.ConnectionID, string(msgJSON))
        if err != nil {
            log.Printf("Failed to send to connection %s: %v", conn.ConnectionID, err)
            sendErr = err // Keep last error but continue trying other connections
        }
    }
    
    return sendErr
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
    filterCondition = expression.Name("user_id").Equal(expression.Value(userIDs[0]))
    
    // Add OR conditions for each additional user ID
    for i := 1; i < len(userIDs); i++ {
        filterCondition = filterCondition.Or(expression.Name("user_id").Equal(expression.Value(userIDs[i])))
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



// handleSubscribeAction processes WebSocket subscription requests
func handleSubscribeAction(req events.APIGatewayWebsocketProxyRequest, data json.RawMessage) (events.APIGatewayProxyResponse, error) {
    // Parse the subscription data
    var subRequest struct {
        Country string `json:"country"`
        Region  string `json:"region"`
        Spot    string `json:"spot"`
    }

    if err := json.Unmarshal(data, &subRequest); err != nil {
        log.Printf("Error parsing subscription data: %v", err)
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusBadRequest,
            Body:       "Invalid subscription data",
        }, nil
    }

    connectionID := req.RequestContext.ConnectionID

    // Get the connection info to know which user is subscribing
    connection, err := getConnection(connectionID)
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
    input := &dynamodb.PutItemInput{
        TableName: aws.String("SpotSubscriptions"),
        Item: map[string]*dynamodb.AttributeValue{
            "spot_id": {
                S: aws.String(spotIdentifier),
            },
            "user_id": {
                S: aws.String(userID),
            },
            "subscribed_at": {
                S: aws.String(time.Now().Format(time.RFC3339)),
            },
            "connection_id": {
                S: aws.String(connectionID),
            },
        },
    }

    _, err = db.PutItem(input)
    if err != nil {
        log.Printf("Failed to save subscription: %v", err)
        return events.APIGatewayProxyResponse{
            StatusCode: http.StatusInternalServerError,
            Body:       "Failed to subscribe",
        }, nil
    }

    // Update connection metadata with current spot
    updateConnectionSpot(connectionID, spotIdentifier)

    // Send confirmation back to client
    response := map[string]interface{}{
        "action": "subscribed",
        "data": map[string]interface{}{
            "spot_id": spotIdentifier,
            "success": true,
        },
    }

    responseJSON, _ := json.Marshal(response)
    sendToConnection(connectionID, string(responseJSON))

    log.Printf("User %s subscribed to spot: %s via connection %s", userID, spotIdentifier, connectionID)
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
        Body:       "Subscribed",
    }, nil
}

// handlePingAction responds to ping messages to keep the connection alive
func handlePingAction(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    connectionID := req.RequestContext.ConnectionID
    
    // Update last active time
    updateConnectionLastActive(connectionID)
    
    // Send pong response
    response := map[string]interface{}{
        "action": "pong",
        "data": map[string]interface{}{
            "time": time.Now().Format(time.RFC3339),
        },
    }
    
    responseJSON, _ := json.Marshal(response)
    sendToConnection(connectionID, string(responseJSON))
    
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
        Body:       "Ping received",
    }, nil
}

// getConnection retrieves connection info from DynamoDB
func getConnection(connectionID string) (*ConnectionInfo, error) {
    input := &dynamodb.GetItemInput{
        TableName: aws.String("WebSocketConnections"),
        Key: map[string]*dynamodb.AttributeValue{
            "connection_id": {
                S: aws.String(connectionID),
            },
        },
    }

    result, err := db.GetItem(input)
    if err != nil {
        return nil, err
    }

    if len(result.Item) == 0 {
        return nil, fmt.Errorf("connection not found")
    }

    var connection ConnectionInfo
    err = dynamodbattribute.UnmarshalMap(result.Item, &connection)
    if err != nil {
        return nil, err
    }

    return &connection, nil
}

// updateConnectionSpot updates the current spot for a connection
func updateConnectionSpot(connectionID string, spotIdentifier string) error {
    newTTL := time.Now().Add(24 * time.Hour).Unix()

	input := &dynamodb.UpdateItemInput{
        TableName: aws.String("WebSocketConnections"),
        Key: map[string]*dynamodb.AttributeValue{
            "connection_id": {
                S: aws.String(connectionID),
            },
        },
        UpdateExpression: aws.String("SET CurrentSpot = :spot, LastActive = :time"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":spot": {
                S: aws.String(spotIdentifier),
            },
            ":time": {
                S: aws.String(time.Now().Format(time.RFC3339)),
            },
			":ttl": {
                N: aws.String(fmt.Sprintf("%d", newTTL)),
            },
        },
    }

    _, err := db.UpdateItem(input)
    return err
}