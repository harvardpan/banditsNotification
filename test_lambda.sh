#!/bin/bash

# Test script for local Lambda function
set -e

LAMBDA_URL="http://localhost:9000/2015-03-31/functions/function/invocations"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Test payload
TEST_PAYLOAD='{
    "source": "test",
    "detail-type": "Manual Test",
    "detail": {
        "test": true
    }
}'

print_status "Testing Lambda function at $LAMBDA_URL"
print_status "Payload: $TEST_PAYLOAD"

# Check if lambda is running
if ! curl -s $LAMBDA_URL > /dev/null 2>&1; then
    print_error "Lambda function is not running at $LAMBDA_URL"
    print_error "Please start it with: make lambda-local"
    exit 1
fi

print_status "Lambda is running. Sending test request..."

# Send test request
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$TEST_PAYLOAD" \
    $LAMBDA_URL)

# Check if request was successful
if [ $? -eq 0 ]; then
    print_status "Request sent successfully!"
    echo "Response:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
else
    print_error "Failed to send request to Lambda function"
    exit 1
fi