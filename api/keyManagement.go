package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// APIKey represents an API key in the system
type APIKey struct {
    KeyID       string    `json:"key_id"`
    KeyValue    string    `json:"key_value"`
    Description string    `json:"description"`
    CreatedBy   string    `json:"created_by"`
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    Scopes      []string  `json:"scopes"`
}

// Create a new API key with specified parameters
func GenerateAPIKey(description string, createdBy string, durationDays int, scopes []string) (*APIKey, error) {
    // Generate a random key
    b := make([]byte, 32)
    _, err := rand.Read(b)
    if err != nil {
        return nil, err
    }
    
    // Generate a unique ID
    idBytes := make([]byte, 16)
    _, err = rand.Read(idBytes)
    if err != nil {
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
    
    apiKey := &APIKey{
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

// Store an API key in DynamoDB
func storeAPIKey(apiKey *APIKey) error {
    item, err := dynamodbattribute.MarshalMap(apiKey)
    if err != nil {
        return err
    }
    
    input := &dynamodb.PutItemInput{
        TableName: aws.String("ApiKeys"),
        Item:      item,
    }
    
    _, err = db.PutItem(input)
    return err
}

// Validate an API key against DynamoDB
func validateAPIKey(keyValue string, requiredScope string) (*APIKey, bool) {
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
    
    result, err := db.Scan(input)
    if err != nil || len(result.Items) == 0 {
        fmt.Print("nothing found for key")
        return nil, false
    }
    
    var apiKey APIKey
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
    if requiredScope != "" {
        hasScope := false
        for _, scope := range apiKey.Scopes {
            if scope == requiredScope {
                hasScope = true
                break
            }
        }
        if !hasScope {
            fmt.Print("wrong scope")
            return nil, false
        }
    }
    
    return &apiKey, true
}

func isAdminUser(email string) bool {
    adminUsers := map[string]bool{
        "ryancpatton0@gmail.com": true,
    }
    
    return adminUsers[email]
}