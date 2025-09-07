#!/bin/bash

# Deploy script for Bandits Notification Lambda using AWS SSO
# This script requires AWS SSO to be configured and logged in
set -e

# Configuration
STACK_NAME="bandits-notification-v3"
AWS_REGION="us-east-1"
ECR_REPOSITORY="development/bandits-notification"
KMS_KEY_ARN="arn:aws:kms:us-east-1:028036396420:alias/BanditsNotifierKMSKey"
SSO_PROFILE="developmentadmin"  # Your SSO profile name with admin access

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if AWS SSO is logged in
print_status "Checking AWS SSO login status..."
if ! AWS_PROFILE=$SSO_PROFILE aws sts get-caller-identity &> /dev/null; then
    print_warning "AWS SSO session expired. Logging in..."
    aws sso login --profile $SSO_PROFILE
fi

# Export the profile for all subsequent commands
export AWS_PROFILE=$SSO_PROFILE

print_status "Using AWS Profile: $AWS_PROFILE"

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}"

print_status "Using AWS Account: ${AWS_ACCOUNT_ID}"
print_status "Deploying to region: ${AWS_REGION}"
print_status "Stack name: ${STACK_NAME}"

# Create ECR repository if it doesn't exist
print_status "Creating ECR repository if it doesn't exist..."
aws ecr describe-repositories --repository-names ${ECR_REPOSITORY} --region ${AWS_REGION} &> /dev/null || \
aws ecr create-repository --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION} \
  --image-scanning-configuration scanOnPush=true

# Login to ECR
print_status "Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Build and push Docker image
print_status "Building Lambda Docker image..."
docker buildx build --platform linux/amd64 --provenance=false -f Dockerfile.lambda -t ${ECR_URI}:latest .

print_status "Pushing Docker image to ECR..."
docker push ${ECR_URI}:latest

# Deploy CloudFormation stack
print_status "Deploying CloudFormation stack..."
aws cloudformation deploy \
    --template-file infrastructure/lambda-stack.yaml \
    --stack-name ${STACK_NAME} \
    --parameter-overrides \
        ECRRepositoryURI=${ECR_URI} \
        ImageTag=latest \
        S3BucketName=banditsnotifier-storage \
        KMSKeyArn=${KMS_KEY_ARN} \
    --capabilities CAPABILITY_NAMED_IAM \
    --region ${AWS_REGION}

# Get the Lambda function name
LAMBDA_FUNCTION_NAME=$(aws cloudformation describe-stacks \
    --stack-name ${STACK_NAME} \
    --region ${AWS_REGION} \
    --query 'Stacks[0].Outputs[?OutputKey==`LambdaFunctionName`].OutputValue' \
    --output text)

print_status "Lambda function created: ${LAMBDA_FUNCTION_NAME}"

# Update Lambda function code
print_status "Updating Lambda function code..."
aws lambda update-function-code \
    --function-name ${LAMBDA_FUNCTION_NAME} \
    --image-uri ${ECR_URI}:latest \
    --region ${AWS_REGION}

# Wait for update to complete
print_status "Waiting for Lambda update to complete..."
aws lambda wait function-updated \
    --function-name ${LAMBDA_FUNCTION_NAME} \
    --region ${AWS_REGION}

print_status "Deployment completed successfully!"

# Show stack outputs
print_status "Stack outputs:"
aws cloudformation describe-stacks \
    --stack-name ${STACK_NAME} \
    --region ${AWS_REGION} \
    --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
    --output table

print_warning "SSO session will expire. Re-run 'aws sso login --profile $SSO_PROFILE' when needed."