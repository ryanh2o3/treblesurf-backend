#!/bin/bash

set -e

# Define variables
FUNCTION_NAME="api"
REGION="eu-west-1"
ZIP_FILE="function.zip"

# Build the Go application
GOOS=linux GOARCH=amd64 go build

ls

# Package the application
zip $ZIP_FILE main.go