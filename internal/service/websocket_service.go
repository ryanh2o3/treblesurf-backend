package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/golang-jwt/jwt"
)

type WebSocketService struct {
	dbStorage storage.DynamoDBStorage
	jwtSecret []byte
}

func NewWebSocketService(dbStorage storage.DynamoDBStorage, jwtSecret []byte) *WebSocketService {
	return &WebSocketService{
		dbStorage: dbStorage,
		jwtSecret: jwtSecret,
	}
}

func (s *WebSocketService) ValidateWebSocketToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
}

func (s *WebSocketService) SaveConnection(conn *model.ConnectionInfo) error {
	conn.TTL = time.Now().Add(24 * time.Hour).Unix()

	item, err := dynamodbattribute.MarshalMap(conn)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("WebSocketConnections"),
		Item:      item,
	}

	_, err = s.dbStorage.PutItem(input)
	return err
}

func (s *WebSocketService) DeleteConnection(connectionID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("WebSocketConnections"),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {
				S: aws.String(connectionID),
			},
		},
	}

	_, err := s.dbStorage.DeleteItem(input)
	return err
}

func (s *WebSocketService) UpdateConnectionLastActive(connectionID string) error {
	newTTL := time.Now().Add(24 * time.Hour).Unix()

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("WebSocketConnections"),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {
				S: aws.String(connectionID),
			},
		},
		UpdateExpression: aws.String("SET LastActive = :time, ttl = :ttl"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":time": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
			":ttl": {
				N: aws.String(fmt.Sprintf("%d", newTTL)),
			},
		},
	}

	_, err := s.dbStorage.UpdateItem(input)
	return err
}

func (s *WebSocketService) GetConnection(connectionID string) (*model.ConnectionInfo, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String("WebSocketConnections"),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {
				S: aws.String(connectionID),
			},
		},
	}

	result, err := s.dbStorage.GetItem(input)
	if err != nil {
		return nil, err
	}

	if len(result.Item) == 0 {
		return nil, fmt.Errorf("connection not found")
	}

	var connection model.ConnectionInfo
	err = dynamodbattribute.UnmarshalMap(result.Item, &connection)
	if err != nil {
		return nil, err
	}

	return &connection, nil
}

func (s *WebSocketService) UpdateConnectionSpot(connectionID, spotIdentifier string) error {
	newTTL := time.Now().Add(24 * time.Hour).Unix()

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("WebSocketConnections"),
		Key: map[string]*dynamodb.AttributeValue{
			"connection_id": {
				S: aws.String(connectionID),
			},
		},
		UpdateExpression: aws.String("SET CurrentSpot = :spot, LastActive = :time, ttl = :ttl"),
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

	_, err := s.dbStorage.UpdateItem(input)
	return err
}

func (s *WebSocketService) SaveSubscription(spotIdentifier, userID, connectionID string) error {
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

	_, err := s.dbStorage.PutItem(input)
	return err
}

// SendToConnection sends a message to a specific WebSocket client
func (s *WebSocketService) SendToConnection(connectionID, message string) error {
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

func (s *WebSocketService) GetSourceIP(headers map[string]string) string {
	// Try CloudFront-specific header first
	if ip, ok := headers["CloudFront-Viewer-Address"]; ok && ip != "" {
		return strings.Split(ip, ":")[0]
	}

	// Try standard headers
	if ip, ok := headers["X-Forwarded-For"]; ok && ip != "" {
		// X-Forwarded-For may contain multiple IPs, take the first one
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fallback to source IP from API Gateway
	if ip, ok := headers["X-Forwarded-For"]; ok && ip != "" {
		return ip
	}

	return "unknown"
}

func (s *WebSocketService) CreateSubscriptionResponse(spotIdentifier string) *model.WebSocketResponse {
	return &model.WebSocketResponse{
		Action: "subscribed",
		Data: map[string]interface{}{
			"spot_id": spotIdentifier,
			"success": true,
		},
	}
}

func (s *WebSocketService) CreatePongResponse() *model.WebSocketResponse {
	return &model.WebSocketResponse{
		Action: "pong",
		Data: map[string]interface{}{
			"time": time.Now().Format(time.RFC3339),
		},
	}
}

// BroadcastToUsers sends a message to multiple users via their WebSocket connections
func (s *WebSocketService) BroadcastToUsers(userIDs []string, message interface{}) error {
	// Get all connections for the given user IDs
	connections, err := s.GetConnectionsByUserIDs(userIDs)
	if err != nil {
		return fmt.Errorf("failed to get connections: %v", err)
	}

	// Send message to each connection
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	for _, conn := range connections {
		if err := s.SendToConnection(conn.ConnectionID, string(messageJSON)); err != nil {
			log.Printf("Failed to send message to connection %s: %v", conn.ConnectionID, err)
			// Continue with other connections
		}
	}

	return nil
}

func (s *WebSocketService) GetConnectionsByUserIDs(userIDs []string) ([]*model.ConnectionInfo, error) {
	var allConnections []*model.ConnectionInfo

	for _, userID := range userIDs {
		input := &dynamodb.ScanInput{
			TableName:        aws.String("WebSocketConnections"),
			FilterExpression: aws.String("user_id = :user_id"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":user_id": {
					S: aws.String(userID),
				},
			},
		}

		result, err := s.dbStorage.Scan(input)
		if err != nil {
			log.Printf("Error scanning connections for user %s: %v", userID, err)
			continue
		}

		for _, item := range result.Items {
			var conn model.ConnectionInfo
			if err := dynamodbattribute.UnmarshalMap(item, &conn); err != nil {
				log.Printf("Error unmarshalling connection: %v", err)
				continue
			}
			allConnections = append(allConnections, &conn)
		}
	}

	return allConnections, nil
}
