package service

import (
	"treblesurf-backend/internal/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type UserService struct {
	db *dynamodb.DynamoDB
}

func NewUserService(db *dynamodb.DynamoDB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetUserByEmail(email string) (*model.User, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
	}

	result, err := s.db.GetItem(input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var user model.User
	err = dynamodbattribute.UnmarshalMap(result.Item, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) GetUserByUUID(uuid string) (*model.User, error) {
	// Since UUID is not the primary key, we need to scan the table
	// In production, you might want to create a GSI on UUID
	input := &dynamodb.ScanInput{
		TableName:        aws.String("Users"),
		FilterExpression: aws.String("#uuid = :uuid"),
		ExpressionAttributeNames: map[string]*string{
			"#uuid": aws.String("uuid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uuid": {
				S: aws.String(uuid),
			},
		},
	}

	result, err := s.db.Scan(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var user model.User
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) UpdateUserTheme(email, theme string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
		UpdateExpression: aws.String("SET theme = :theme"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":theme": {
				S: aws.String(theme),
			},
		},
	}

	_, err := s.db.UpdateItem(input)
	return err
}

func (s *UserService) DeleteUser(email string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
	}

	_, err := s.db.DeleteItem(input)
	return err
}
