#!/bin/bash
# filepath: local/scripts/setup.sh

# Create necessary directories
mkdir -p ./local/data/dynamodb
mkdir -p ./local/data/s3/treblesurf-images/snapshots
mkdir -p ./local/data/s3/treblesurf-images/surf-reports
mkdir -p ./local/data/localstack

if docker ps | grep -q "treblesurf-localstack" && docker ps | grep -q "dynamodb-local"; then
  echo "Docker services are already running."
else
  # Start the Docker services
  echo "Starting local AWS services..."
  docker-compose up -d

  # Wait for services to be ready
  echo "Waiting for services to start up..."
  # Wait for DynamoDB to be ready
  while ! curl -s http://localhost:8000 > /dev/null; do
    echo "Waiting for DynamoDB to be ready..."
    sleep 2
  done
  
  # Wait for LocalStack to be ready
  while ! curl -s http://localhost:4566/_localstack/health | grep -q '"s3": "running"'; do
    echo "Waiting for LocalStack to be ready..."
    sleep 2
  done
fi

# Create the S3 bucket
echo "Creating S3 bucket in LocalStack..."
docker exec treblesurf-localstack awslocal s3 mb s3://treblesurf-images

# Seed the database
echo "Seeding DynamoDB with test data..."
go run ./seed_data.go

echo "Local development environment is ready!"
echo "Run 'go run ./local/cmd/server.go' to start the local server"