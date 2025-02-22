#!/bin/bash

set -e

# Define variables
FUNCTION_NAME="your_lambda_function_name"
REGION="your_aws_region"
ZIP_FILE="function.zip"
SOURCE_DIR="function"

# Build the Go application
GOOS=linux GOARCH=amd64 go build -o main $SOURCE_DIR/main.go

# Package the application
zip $ZIP_FILE main

# Deploy to AWS Lambda
aws lambda update-function-code --function-name $FUNCTION_NAME --zip-file fileb://$ZIP_FILE --region $REGION

# Clean up
rm main
rm $ZIP_FILE

echo "Deployment to AWS Lambda completed successfully."