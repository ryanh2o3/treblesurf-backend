#!/bin/bash

set -e

# Define variables
FUNCTION_NAME="api"
REGION="eu-west-1"
ZIP_FILE="function.zip"

# Build the Go application
GOOS=linux GOARCH=amd64 go build

# Package the application
zip $ZIP_FILE main

# Deploy to AWS Lambda
aws lambda update-function-code --function-name $FUNCTION_NAME --zip-file fileb://$ZIP_FILE --region $REGION

# Clean up
rm main
rm $ZIP_FILE

echo "Deployment to AWS Lambda completed successfully."