# AWS Lambda Deployment Guide

This guide explains how to deploy the Bandits Notification system to AWS Lambda using containers.

## Architecture

- **AWS Lambda**: Runs the Go application in a container
- **EventBridge**: Schedules the Lambda to run every 5 minutes
- **ECR**: Stores container images
- **S3**: Stores schedule data and screenshots
- **CloudWatch**: Logs and monitoring
- **GitHub Actions**: CI/CD pipeline

## Prerequisites

1. **AWS CLI configured** with appropriate permissions
2. **Docker** installed and running
3. **Go 1.21+** installed
4. **SOPS** configured with KMS key access
5. **GitHub repository** set up

## Initial Setup

### 1. Configure AWS Permissions

Your AWS user/role needs the following permissions:
- CloudFormation full access
- Lambda full access
- ECR full access
- S3 full access
- EventBridge full access
- IAM role creation
- KMS decrypt access for SOPS

### 2. Set up KMS Key

Ensure the KMS key exists and you have access:
```bash
aws kms describe-key --key-id arn:aws:kms:us-east-1:028036396420:alias/BanditsNotifierKMSKey
```

### 3. Prepare Configuration

1. Copy and configure your secrets:
   ```bash
   cp test_config.yaml.example secrets.yaml
   # Edit secrets.yaml with your actual credentials
   ```

2. Encrypt the secrets file:
   ```bash
   make encrypt
   ```

## Local Development

### Build and Test Locally

```bash
# Build the application
make build

# Run tests
make test

# Build Docker image
make docker-build

# Test Lambda function locally
make lambda-local
```

### Testing Local Lambda

1. Start the local Lambda runtime:
   ```bash
   make lambda-local
   ```

2. In another terminal, test the function:
   ```bash
   curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" \
     -d '{"source":"test","detail-type":"Manual Test","detail":{"test":true}}'
   ```

## Deployment

### Option 1: Manual Deployment

```bash
# Deploy everything (infrastructure + application)
make lambda-deploy
```

This script will:
1. Create ECR repository
2. Build and push Docker image
3. Deploy CloudFormation stack
4. Update Lambda function

### Option 2: CI/CD with GitHub Actions

1. **Set up GitHub Secrets:**
   ```
   AWS_ACCESS_KEY_ID=your_access_key
   AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

2. **Push to main branch:**
   ```bash
   git add .
   git commit -m "Deploy Lambda container setup"
   git push origin main
   ```

3. **Monitor deployment:**
   - Check GitHub Actions tab for build status
   - Check AWS CloudFormation console for stack deployment
   - Check AWS Lambda console for function updates

## Configuration

### Environment Variables

The Lambda function uses these environment variables:

- `CONFIG_PATH`: Path to the encrypted configuration file
- `S3_BUCKET`: S3 bucket name for data storage
- `AWS_REGION`: AWS region

### CloudFormation Parameters

Key parameters in `infrastructure/lambda-stack.yaml`:

- `ECRRepositoryURI`: ECR repository for container images
- `ImageTag`: Container image tag (defaults to latest)
- `S3BucketName`: S3 bucket for data storage
- `KMSKeyArn`: KMS key for SOPS decryption

## Monitoring

### CloudWatch Logs

View Lambda logs:
```bash
aws logs tail /aws/lambda/bandits-notification-prod-bandits-notification --follow
```

### Lambda Metrics

Monitor in AWS Console:
- Lambda → Functions → bandits-notification-prod-bandits-notification
- CloudWatch → Metrics → AWS/Lambda

### EventBridge Schedule

View scheduled executions:
- EventBridge → Rules → bandits-notification-prod-schedule

## Troubleshooting

### Common Issues

1. **Container build fails:**
   ```bash
   # Check Go modules and dependencies
   go mod tidy
   go mod download
   ```

2. **Lambda timeout:**
   - Default timeout is 15 minutes
   - Increase in CloudFormation template if needed

3. **Permission errors:**
   ```bash
   # Check IAM role permissions
   aws iam get-role --role-name bandits-notification-prod-lambda-execution-role
   ```

4. **SOPS decryption fails:**
   - Verify KMS key permissions
   - Check if secrets.yaml is properly encrypted

### Debug Lambda Function

1. **Manual invocation:**
   ```bash
   aws lambda invoke \
     --function-name bandits-notification-v3-bandits-notification \
     --payload '{"source":"manual","detail-type":"Debug","detail":{}}' \
     --cli-binary-format raw-in-base64-out \
     response.json
   
   cat response.json
   ```

2. **Check logs:**
   ```bash
   aws logs filter-log-events \
     --log-group-name /aws/lambda/bandits-notification-v3-bandits-notification \
     --start-time $(date -d '5 minutes ago' +%s)000
   ```

## Scaling and Performance

### Memory and Timeout

Adjust in CloudFormation template:
```yaml
MemorySize: 1024  # MB (128-10240)
Timeout: 900      # seconds (max 15 minutes)
```

### Concurrency

Lambda handles concurrent executions automatically. For this use case (scheduled every 5 minutes), concurrency is not a concern.

### Cost Optimization

- Function runs every 5 minutes = ~8,640 invocations/month
- With 1GB memory and ~30-60 second runtime
- Estimated cost: $2-5/month

## Security

### Encryption

- All secrets encrypted with SOPS/KMS
- S3 bucket encrypted at rest
- Lambda logs encrypted in CloudWatch

### Network Security

- Lambda runs in AWS managed VPC
- No inbound network access
- Outbound access for Twitter API and website scraping

### IAM Permissions

Follows least privilege principle:
- Lambda can only access required S3 bucket
- KMS decrypt permissions only for specific key
- No unnecessary permissions granted

## Backup and Recovery

### S3 Data

- S3 versioning enabled
- Previous schedules stored in archive/
- Screenshots archived with timestamps

### Lambda Function

- Container images stored in ECR
- CloudFormation template in version control
- Infrastructure as code for easy recreation

## Updates and Maintenance

### Updating the Application

1. **Via GitHub Actions (recommended):**
   - Push changes to main branch
   - CI/CD automatically builds and deploys

2. **Manual update:**
   ```bash
   make lambda-deploy
   ```

### Updating Infrastructure

1. Modify `infrastructure/lambda-stack.yaml`
2. Deploy changes:
   ```bash
   aws cloudformation deploy \
     --template-file infrastructure/lambda-stack.yaml \
     --stack-name bandits-notification-prod \
     --capabilities CAPABILITY_NAMED_IAM
   ```

### Updating Dependencies

```bash
go get -u ./...
go mod tidy
```

Then redeploy using either method above.