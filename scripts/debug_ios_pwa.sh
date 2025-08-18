#!/bin/bash

# Debug script for iOS PWA report submission issues
# This script helps identify the specific problem with iOS PWA requests

echo "=== iOS PWA Report Submission Debug Script ==="
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "Error: Please run this script from the project root directory"
    exit 1
fi

echo "1. Checking current environment..."
echo "   GO_ENV: ${GO_ENV:-not set}"
echo "   Current directory: $(pwd)"
echo ""

echo "2. Checking if local server is running..."
if curl -s http://localhost:8080/regions > /dev/null 2>&1; then
    echo "   ✓ Local server is running on port 8080"
else
    echo "   ✗ Local server is not running on port 8080"
    echo "   Please start the local server first:"
    echo "   cd local && go run cmd/server.go"
    echo ""
    exit 1
fi

echo ""
echo "3. Testing basic connectivity..."
echo "   Testing /regions endpoint..."
response=$(curl -s -w "HTTP_STATUS:%{http_code}" http://localhost:8080/regions)
http_status=$(echo "$response" | grep -o 'HTTP_STATUS:[0-9]*' | cut -d: -f2)
if [ "$http_status" = "200" ]; then
    echo "   ✓ /regions endpoint working (HTTP $http_status)"
else
    echo "   ✗ /regions endpoint failed (HTTP $http_status)"
fi

echo ""
echo "4. Testing authentication flow..."
echo "   Note: This requires a valid Google ID token"
echo "   You can test with a mock token in development mode"
echo ""

echo "5. Testing report submission endpoint..."
echo "   Testing /submitSurfReport endpoint..."
echo "   Note: This requires authentication"

# Test with a mock request (will fail without auth, but shows the endpoint exists)
mock_response=$(curl -s -w "HTTP_STATUS:%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1" \
    -d '{"country":"Ireland","region":"Donegal","spot":"Ballymastocker","surfSize":"head-high","windAmount":"light","windDirection":"onshore","quality":"hollow","consistency":"setty","messiness":"messy","imageData":""}' \
    http://localhost:8080/submitSurfReport)

mock_http_status=$(echo "$mock_response" | grep -o 'HTTP_STATUS:[0-9]*' | cut -d: -f2)
echo "   Mock request response: HTTP $mock_http_status"

if [ "$mock_http_status" = "401" ]; then
    echo "   ✓ Endpoint exists but requires authentication (expected)"
elif [ "$mock_http_status" = "200" ]; then
    echo "   ⚠ Endpoint returned 200 (might be in dev mode with mock auth)"
else
    echo "   ✗ Unexpected response: HTTP $mock_http_status"
fi

echo ""
echo "6. Common iOS PWA issues to check:"
echo "   - Cookie settings (Secure flag, SameSite, HTTPOnly)"
echo "   - CORS configuration for production"
echo "   - CSRF token handling"
echo "   - Session cookie persistence"
echo "   - HTTPS vs HTTP cookie handling"
echo ""

echo "7. Next steps:"
echo "   - Check server logs for detailed error messages"
echo "   - Verify iOS PWA is sending proper headers"
echo "   - Test with different iOS versions"
echo "   - Check if cookies are being set properly"
echo ""

echo "=== Debug script completed ==="
