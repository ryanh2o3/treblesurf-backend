#!/bin/bash
# One-time AWS setup for iOS push notifications.
# Requires AWS CLI credentials for account 759663378274, region eu-west-1.
#
# Prerequisites (outside this script):
#   1. Apple Developer: enable Push Notifications for treble.TrebleSurf
#   2. Create an APNs Auth Key (.p8) and note Key ID + Team ID
#   3. Store GitHub Actions secrets: APNS_KEY_P8, APNS_KEY_ID, APNS_TEAM_ID
#   4. Xcode: Push Notifications capability (aps-environment in entitlements)

set -euo pipefail

REGION="${AWS_REGION:-eu-west-1}"
ACCOUNT_ID="${AWS_ACCOUNT_ID:-759663378274}"
ROLE_ARN="${NOTIFICATIONS_LAMBDA_ROLE_ARN:-arn:aws:iam::${ACCOUNT_ID}:role/service-role/api-role-pydrwpue}"
FUNCTION_NAME="${NOTIFICATIONS_FUNCTION_NAME:-notifications}"

echo "Creating DynamoDB tables in ${REGION} (idempotent)..."

create_table_if_missing() {
  local name="$1"
  if aws dynamodb describe-table --region "$REGION" --table-name "$name" >/dev/null 2>&1; then
    echo "Table ${name} already exists"
    return 0
  fi
  shift
  aws dynamodb create-table --region "$REGION" "$@"
  aws dynamodb wait table-exists --region "$REGION" --table-name "$name"
  echo "Created table ${name}"
}

create_table_if_missing DeviceTokens \
  --table-name DeviceTokens \
  --attribute-definitions \
    AttributeName=user_uuid,AttributeType=S \
    AttributeName=token,AttributeType=S \
  --key-schema \
    AttributeName=user_uuid,KeyType=HASH \
    AttributeName=token,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST

create_table_if_missing SpotAlertSubscriptions \
  --table-name SpotAlertSubscriptions \
  --attribute-definitions \
    AttributeName=spot_id,AttributeType=S \
    AttributeName=user_uuid,AttributeType=S \
  --key-schema \
    AttributeName=spot_id,KeyType=HASH \
    AttributeName=user_uuid,KeyType=RANGE \
  --global-secondary-indexes '[
    {
      "IndexName": "user_uuid-index",
      "KeySchema": [
        {"AttributeName": "user_uuid", "KeyType": "HASH"},
        {"AttributeName": "spot_id", "KeyType": "RANGE"}
      ],
      "Projection": {"ProjectionType": "ALL"}
    }
  ]' \
  --billing-mode PAY_PER_REQUEST

if aws lambda get-function --region "$REGION" --function-name "$FUNCTION_NAME" >/dev/null 2>&1; then
  echo "Lambda ${FUNCTION_NAME} already exists"
else
  echo "Creating placeholder Lambda ${FUNCTION_NAME} (deploy.yml updates the code)..."
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  cat > "${tmpdir}/main.go" <<'EOF'
package main

import "github.com/aws/aws-lambda-go/lambda"

func main() { lambda.Start(func() error { return nil }) }
EOF
  (cd "$tmpdir" && GOOS=linux GOARCH=amd64 go mod init placeholder && go get github.com/aws/aws-lambda-go@v1.47.0 && go build -o bootstrap main.go && zip bootstrap.zip bootstrap)
  aws lambda create-function \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --runtime provided.al2023 \
    --handler bootstrap \
    --architectures x86_64 \
    --role "$ROLE_ARN" \
    --zip-file "fileb://${tmpdir}/bootstrap.zip" \
    --timeout 60 \
    --memory-size 256
  echo "Created Lambda ${FUNCTION_NAME}"
fi

RULE_NAME="${NOTIFICATIONS_EVENT_RULE:-notifications-hourly}"
if aws events describe-rule --region "$REGION" --name "$RULE_NAME" >/dev/null 2>&1; then
  echo "EventBridge rule ${RULE_NAME} already exists"
else
  aws events put-rule \
    --region "$REGION" \
    --name "$RULE_NAME" \
    --schedule-expression "rate(1 hour)" \
    --state ENABLED
  echo "Created EventBridge rule ${RULE_NAME}"
fi

FUNCTION_ARN="arn:aws:lambda:${REGION}:${ACCOUNT_ID}:function:${FUNCTION_NAME}"
aws events put-targets \
  --region "$REGION" \
  --rule "$RULE_NAME" \
  --targets "Id=notifications,Arn=${FUNCTION_ARN}"

aws lambda add-permission \
  --region "$REGION" \
  --function-name "$FUNCTION_NAME" \
  --statement-id AllowEventBridgeHourly \
  --action lambda:InvokeFunction \
  --principal events.amazonaws.com \
  --source-arn "arn:aws:events:${REGION}:${ACCOUNT_ID}:rule/${RULE_NAME}" \
  2>/dev/null || echo "EventBridge invoke permission already present"

echo "Notifications infrastructure is ready."
