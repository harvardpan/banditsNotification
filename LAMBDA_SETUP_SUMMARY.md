# AWS Lambda Container Deployment - Setup Summary

## 🎉 What's Been Created

Your Bandits Notification system is now ready for AWS Lambda deployment with containers! Here's everything that was set up:

## 📁 New Files Created

### 🐳 Container Setup
- **`Dockerfile`** - Production Lambda container image
- **`Dockerfile.local`** - Local development with Lambda Runtime Interface Emulator
- **`docker-compose.yml`** - Local development environment
- **`cmd/lambda/main.go`** - Lambda handler function

### 🏗️ Infrastructure 
- **`infrastructure/lambda-stack.yaml`** - CloudFormation template for complete AWS infrastructure
- **`infrastructure/deploy.sh`** - Manual deployment script

### 🚀 CI/CD
- **`.github/workflows/deploy.yml`** - GitHub Actions workflow for automated deployment

### 📖 Documentation
- **`DEPLOYMENT.md`** - Complete deployment and operations guide
- **`LAMBDA_SETUP_SUMMARY.md`** - This summary

### 🛠️ Updated Files
- **`Makefile`** - Added Docker and Lambda targets
- **`go.mod`** - Added AWS Lambda Go SDK dependency

## 🏛️ Infrastructure Components

### AWS Resources Created:
- **Lambda Function** - Containerized Go application 
- **ECR Repository** - For storing container images
- **S3 Bucket** - For schedule data and screenshots
- **EventBridge Rule** - Triggers Lambda every 5 minutes
- **IAM Roles** - Least privilege permissions
- **CloudWatch Logs** - Function logging with 30-day retention
- **SQS Dead Letter Queue** - For failed executions

## ⚙️ Key Features

### 🔒 Security
- All secrets encrypted with SOPS using your KMS key
- Least privilege IAM permissions
- S3 bucket encryption at rest
- No inbound network access

### 📈 Scalability & Reliability
- Auto-scaling Lambda (handles concurrent executions)
- Dead letter queue for failed executions
- Container image versioning in ECR
- Infrastructure as code with CloudFormation

### 🔄 CI/CD Pipeline
- Automated testing on pull requests
- Security vulnerability scanning
- Automatic deployment on push to main/develop
- Separate prod and dev environments

### 🖥️ Local Development
- Lambda Runtime Interface Emulator for local testing
- Docker Compose for development environment
- Make targets for common tasks

## 🚀 Quick Start Commands

```bash
# Local development
make docker-build          # Build container image
make lambda-local          # Test Lambda locally
make test                  # Run all tests

# Manual deployment
make lambda-deploy         # Deploy to AWS

# Configuration management
make encrypt              # Encrypt secrets.yaml
make decrypt             # Decrypt for editing
```

## 📋 Next Steps

### 1. Set up GitHub Secrets
Add these to your GitHub repository secrets:
```
AWS_ACCESS_KEY_ID=your_access_key_id
AWS_SECRET_ACCESS_KEY=your_secret_access_key
```

### 2. Configure Your Secrets
```bash
# Edit your configuration
cp test_config.yaml.example secrets.yaml
# Add your real Twitter credentials, AWS settings, etc.

# Encrypt the file
make encrypt
```

### 3. Deploy

**Option A: Manual Deployment**
```bash
make lambda-deploy
```

**Option B: GitHub Actions (Recommended)**
```bash
git add .
git commit -m "Set up Lambda container deployment"
git push origin main
```

### 4. Monitor
- 📊 **CloudWatch Logs**: `/aws/lambda/bandits-notification-prod-bandits-notification`
- 📈 **Lambda Metrics**: AWS Console → Lambda → Your Function
- ⏰ **Schedule**: EventBridge → Rules (runs every 5 minutes)

## 💰 Cost Estimate

**Expected Monthly Costs:**
- **Lambda**: ~$2-5/month (8,640 executions × 30-60 seconds)
- **S3**: ~$1-2/month (storage for screenshots and schedules)
- **ECR**: ~$0.10/month (container image storage)
- **CloudWatch Logs**: ~$1/month
- **Total**: ~$4-8/month

## 🔧 Architecture Flow

```
GitHub Push → GitHub Actions → Build Container → Push to ECR → 
Update Lambda → EventBridge triggers Lambda every 5 minutes →
Lambda scrapes websites → Detects changes → Posts to Twitter → 
Saves to S3 → Logs to CloudWatch
```

## 🆘 Troubleshooting

### Common Issues:

1. **Build Failures**
   ```bash
   go mod tidy
   make docker-build
   ```

2. **Permission Issues**
   - Verify AWS credentials have CloudFormation, Lambda, ECR, S3 access
   - Check KMS key permissions for SOPS

3. **Lambda Timeouts**
   - Default timeout is 15 minutes
   - Check CloudWatch logs for errors

4. **Schedule Not Running**
   - Verify EventBridge rule is enabled
   - Check Lambda permissions

### Debug Commands:
```bash
# View logs
aws logs tail /aws/lambda/bandits-notification-prod-bandits-notification --follow

# Manual invoke
aws lambda invoke --function-name bandits-notification-prod-bandits-notification --payload '{}' response.json

# Check stack status  
aws cloudformation describe-stacks --stack-name bandits-notification-prod
```

## 🎯 Benefits Achieved

✅ **Serverless**: No servers to manage, auto-scaling
✅ **Cost Effective**: Pay only for execution time
✅ **Reliable**: AWS managed infrastructure, dead letter queues
✅ **Secure**: Encrypted secrets, least privilege access
✅ **Maintainable**: Infrastructure as code, CI/CD pipeline
✅ **Observable**: CloudWatch logs and metrics
✅ **Testable**: Local development environment

## 📚 Additional Resources

- **Full Deployment Guide**: See `DEPLOYMENT.md`
- **AWS Lambda Docs**: https://docs.aws.amazon.com/lambda/
- **Container Images**: https://docs.aws.amazon.com/lambda/latest/dg/images.html
- **EventBridge Scheduling**: https://docs.aws.amazon.com/eventbridge/

---

Your Bandits Notification system is now ready for production deployment on AWS Lambda! 🚀