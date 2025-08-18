package storage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DynamoDBStorage interface {
	Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}

type DynamoDBClient struct {
	client *dynamodb.DynamoDB
}

func NewDynamoDBStorage(region string) (*DynamoDBClient, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	client := dynamodb.New(sess)
	return &DynamoDBClient{client: client}, nil
}

func (d *DynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return d.client.Scan(input)
}

func (d *DynamoDBClient) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	return d.client.Query(input)
}

func (d *DynamoDBClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return d.client.GetItem(input)
}

func (d *DynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return d.client.PutItem(input)
}

func (d *DynamoDBClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return d.client.UpdateItem(input)
}

func (d *DynamoDBClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	return d.client.DeleteItem(input)
}

// GetDynamoDBClient returns the underlying DynamoDB client
func (d *DynamoDBClient) GetDynamoDBClient() *dynamodb.DynamoDB {
	return d.client
}

// Helper function to unmarshal DynamoDB items
func UnmarshalMap(item map[string]*dynamodb.AttributeValue, v interface{}) error {
	return dynamodbattribute.UnmarshalMap(item, v)
}

func UnmarshalListOfMaps(items []map[string]*dynamodb.AttributeValue, v interface{}) error {
	return dynamodbattribute.UnmarshalListOfMaps(items, v)
}
