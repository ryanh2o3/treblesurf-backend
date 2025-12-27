// Package storage provides interfaces and implementations for AWS DynamoDB and S3 storage operations.
package storage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// DynamoDBStorage defines the interface for DynamoDB operations.
type DynamoDBStorage interface {
	Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}

// DynamoDBClient provides DynamoDB operations using the AWS SDK.
type DynamoDBClient struct {
	client *dynamodb.DynamoDB
}

// NewDynamoDBStorage creates a new DynamoDB client for the specified AWS region.
func NewDynamoDBStorage(region string) (*DynamoDBClient, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	client := dynamodb.New(sess)
	return &DynamoDBClient{client: client}, nil
}

// Scan performs a scan operation on the DynamoDB table.
func (d *DynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return d.client.Scan(input)
}

// Query performs a query operation on the DynamoDB table.
func (d *DynamoDBClient) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	return d.client.Query(input)
}

// GetItem retrieves a single item from DynamoDB by its key.
func (d *DynamoDBClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return d.client.GetItem(input)
}

// PutItem stores a new item or replaces an existing item in DynamoDB.
func (d *DynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return d.client.PutItem(input)
}

// UpdateItem modifies an existing item's attributes in DynamoDB.
func (d *DynamoDBClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return d.client.UpdateItem(input)
}

// DeleteItem removes an item from DynamoDB by its key.
func (d *DynamoDBClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	return d.client.DeleteItem(input)
}

// GetDynamoDBClient returns the underlying DynamoDB client
func (d *DynamoDBClient) GetDynamoDBClient() *dynamodb.DynamoDB {
	return d.client
}

// UnmarshalMap unmarshals a DynamoDB item map into a Go struct.
func UnmarshalMap(item map[string]*dynamodb.AttributeValue, v interface{}) error {
	return dynamodbattribute.UnmarshalMap(item, v)
}

// UnmarshalListOfMaps unmarshals a list of DynamoDB item maps into a slice of Go structs.
func UnmarshalListOfMaps(items []map[string]*dynamodb.AttributeValue, v interface{}) error {
	return dynamodbattribute.UnmarshalListOfMaps(items, v)
}
