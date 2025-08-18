package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type APIKeyService struct {
	dbStorage storage.DynamoDBStorage
}

func NewAPIKeyService(dbStorage storage.DynamoDBStorage) *APIKeyService {
	return &APIKeyService{
		dbStorage: dbStorage,
	}
}

// GenerateAPIKey creates a new API key with specified parameters
func (s *APIKeyService) GenerateAPIKey(description string, createdBy string, durationDays int, scopes []string) (*model.APIKey, error) {
	// Generate a random key
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	
	// Generate a unique ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	
	keyID := fmt.Sprintf("key_%s", base64.URLEncoding.EncodeToString(idBytes)[:16])
	keyValue := base64.URLEncoding.EncodeToString(b)
	
	// Create expiration date (or far future if 0 days)
	var expiresAt time.Time
	if durationDays <= 0 {
		expiresAt = time.Now().AddDate(10, 0, 0) // 10 years in the future
	} else {
		expiresAt = time.Now().AddDate(0, 0, durationDays)
	}
	
	apiKey := &model.APIKey{
		KeyID:       keyID,
		KeyValue:    keyValue,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Scopes:      scopes,
	}
	
	return apiKey, nil
}

// StoreAPIKey stores an API key in DynamoDB
func (s *APIKeyService) StoreAPIKey(apiKey *model.APIKey) error {
	item, err := dynamodbattribute.MarshalMap(apiKey)
	if err != nil {
		return err
	}
	
	input := &dynamodb.PutItemInput{
		TableName: aws.String("ApiKeys"),
		Item:      item,
	}
	
	_, err = s.dbStorage.PutItem(input)
	return err
}

// ValidateAPIKey validates an API key against DynamoDB
func (s *APIKeyService) ValidateAPIKey(keyValue string, requiredScope string) (*model.APIKey, bool) {
	// Query by key value
	input := &dynamodb.ScanInput{
		TableName:        aws.String("ApiKeys"),
		FilterExpression: aws.String("key_value = :keyValue"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":keyValue": {
				S: aws.String(keyValue),
			},
		},
	}
	
	result, err := s.dbStorage.Scan(input)
	if err != nil || len(result.Items) == 0 {
		fmt.Print("nothing found for key")
		return nil, false
	}
	
	var apiKey model.APIKey
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &apiKey)
	if err != nil {
		fmt.Print("error unmarshalling")
		return nil, false
	}
	
	// Check if the key is expired
	if time.Now().After(apiKey.ExpiresAt) {
		fmt.Print("key expired")
		return nil, false
	}
	
	// Check if the key has the required scope
	for _, scope := range apiKey.Scopes {
		if scope == requiredScope {
			return &apiKey, true
		}
	}
	
	return nil, false
}

// ListAPIKeys retrieves all API keys for a user
func (s *APIKeyService) ListAPIKeys(createdBy string) ([]*model.APIKey, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String("ApiKeys"),
		FilterExpression: aws.String("created_by = :createdBy"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":createdBy": {
				S: aws.String(createdBy),
			},
		},
	}
	
	result, err := s.dbStorage.Scan(input)
	if err != nil {
		return nil, err
	}
	
	var apiKeys []*model.APIKey
	for _, item := range result.Items {
		var apiKey model.APIKey
		if err := dynamodbattribute.UnmarshalMap(item, &apiKey); err != nil {
			continue
		}
		apiKeys = append(apiKeys, &apiKey)
	}
	
	return apiKeys, nil
}

// RevokeAPIKey deletes an API key
func (s *APIKeyService) RevokeAPIKey(keyID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("ApiKeys"),
		Key: map[string]*dynamodb.AttributeValue{
			"key_id": {
				S: aws.String(keyID),
			},
		},
	}
	
	_, err := s.dbStorage.DeleteItem(input)
	return err
}
