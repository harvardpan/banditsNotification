# Bandits Schedule Twitter Notifier - Go Version

This Go application scrapes the Brookline Bandits 12U and 14U web pages and sends out a tweet whenever it detects a difference in the schedule from the last time it checked. The tweet includes a screenshot of the web page captured using headless Chrome.

This is a rewrite of the original Node.js version with improved performance, better security through SOPS encryption, and multiple deployment options including AWS Lambda.

## Prerequisites

1. **Go 1.21+** - This application requires Go 1.21 or later
2. **SOPS** - For encrypted secret management ([Installation Guide](https://github.com/mozilla/sops#download))
3. **Twitter Developer Credentials** - Twitter API access credentials:
   - Consumer Key (API Key)
   - Consumer Secret (API Secret) 
   - Access Token Key
   - Access Token Secret
4. **AWS Credentials** - For S3 storage (optional explicit credentials, can use default credential chain)

## Quick Start

1. **Clone and build:**
```bash
cd banditsNotification
make deps    # Download dependencies
make build   # Build the binary
```

2. **Configure secrets:**
```bash
# Edit secrets.yaml with your credentials
make decrypt                    # Decrypt for editing (if already encrypted)
# Update secrets.yaml with your actual credentials
make encrypt                    # Encrypt with SOPS
```

3. **Run:**
```bash
make run                        # Full operation
make run-dry                    # Dry-run mode (saves files locally)
make run-no-tweet              # Upload to S3 but skip Twitter posts
```

## Configuration

### Secrets Management with SOPS

The application uses SOPS-encrypted `secrets.yaml` for secure credential storage. Example structure:

```yaml
twitter:
    consumer_key: your_twitter_consumer_key
    consumer_secret: your_twitter_consumer_secret
    access_token: your_twitter_access_token
    access_token_secret: your_twitter_access_token_secret
    user_handle: YourTwitterBot

aws:
    # Optional - uses default credential chain if empty
    access_key_id: ""           
    secret_access_key: ""
    region: us-east-1
    s3_bucket: your-s3-bucket-name

app:
    display_timezone: America/New_York
    urls:
        - url: https://your-schedule-url.com
          twitter:
              user_handle: YourTwitterBot
```

### AWS Authentication

The application supports multiple AWS authentication methods:

**Option 1: AWS SSO (Recommended)**
```bash
aws configure sso
aws sso login --profile your-profile
export AWS_PROFILE=your-profile
```

**Option 2: Default Credential Chain**
Automatically uses credentials from:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. `~/.aws/credentials` file
3. IAM roles (for EC2 instances)
4. SSO credentials

**Option 3: Explicit Credentials (Not Recommended)**
Set `access_key_id` and `secret_access_key` in `secrets.yaml`

## Usage

### Development Commands
```bash
make run                        # Run application normally
make run-dry                    # Dry-run mode: save files locally, no S3/Twitter
make run-no-tweet              # No-tweet mode: use S3 but skip Twitter posts
make build                      # Build binary to bin/bandits-notification
```

### Testing
```bash
make test                       # Run all tests
make test-unit                  # Unit tests only (fast)
make test-integration          # Integration tests (requires config)
make test-coverage             # Tests with coverage report
```

### Secret Management
```bash
make decrypt                    # Decrypt secrets.yaml for editing
make encrypt                    # Encrypt secrets.yaml after editing
make decrypt-test              # Decrypt test_config.yaml
make encrypt-test              # Encrypt test_config.yaml
```

### Docker
```bash
make docker-build              # Build Docker image for standalone use
make docker-run                # Run Docker container
make docker-build-lambda       # Build Docker image for AWS Lambda
```

### AWS Lambda
```bash
make lambda-deploy             # Deploy to AWS Lambda
make lambda-invoke             # Test deployed Lambda function
```

## Deployment Options

### 1. Standalone Binary

Build and run directly:
```bash
make build
./bin/bandits-notification
```

### 2. Systemd Service (Linux)

Create `/etc/systemd/system/bandits-notification.service`:
```ini
[Unit]
Description=Bandits Schedule Notification Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/bandits-notification
ExecStart=/path/to/bandits-notification/bin/bandits-notification
Restart=always
RestartSec=300
Environment=CONFIG_PATH=/path/to/secrets.yaml
Environment=AWS_PROFILE=your-profile

[Install]
WantedBy=multi-user.target
```

### 3. macOS LaunchAgent

Create `~/Library/LaunchAgents/com.harvardpan.banditsNotifications.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>com.harvardpan.banditsNotifications</string>
        <key>ProgramArguments</key>
        <array>
            <string>/path/to/bandits-notification/bin/bandits-notification</string>
        </array>
        <key>WorkingDirectory</key>
        <string>/path/to/bandits-notification</string>
        <key>RunAtLoad</key>
        <true/>
        <key>StartInterval</key>
        <integer>300</integer>
        <key>EnvironmentVariables</key>
        <dict>
            <key>CONFIG_PATH</key>
            <string>/path/to/secrets.yaml</string>
            <key>AWS_PROFILE</key>
            <string>your-profile</string>
        </dict>
        <key>StandardOutPath</key>
        <string>/tmp/com.harvardpan.banditsNotifications.out</string>
        <key>StandardErrorPath</key>
        <string>/tmp/com.harvardpan.banditsNotifications.err</string>
    </dict>
</plist>
```

Load with launchctl:
```bash
launchctl bootstrap gui/501 ~/Library/LaunchAgents/com.harvardpan.banditsNotifications.plist
```

### 4. AWS Lambda (Serverless)

Deploy as a serverless function triggered by EventBridge:
```bash
make lambda-deploy
```

The Lambda deployment includes:
- Container image pushed to ECR
- Lambda function with proper IAM roles
- EventBridge rule for scheduled execution
- KMS key for SOPS encryption

## Features

### Processing Modes
- **Normal Mode**: Full operation with S3 uploads and Twitter posts
- **No-Tweet Mode** (`--no-tweet`): Process and upload to S3, skip Twitter
- **Dry-Run Mode** (`--dry-run`): Save files locally, skip S3 and Twitter

### Multi-URL Support
Configure multiple URLs with different Twitter accounts in `secrets.yaml`

### Robust Error Handling
- Detailed logging with timestamps
- Graceful handling of network failures
- AWS credential validation
- Screenshot and parsing error recovery

## Architecture

The application is built with Go using:
- **chromedp**: Headless Chrome for web scraping and screenshots
- **SOPS**: Encrypted secret management 
- **AWS SDK**: S3 integration and Lambda runtime
- **goquery**: HTML parsing for schedule extraction

Key packages:
- `internal/config/`: SOPS configuration management
- `internal/scraper/`: Web scraping with chromedp
- `internal/schedule/`: Schedule parsing and comparison
- `internal/storage/`: AWS S3 integration  
- `internal/twitter/`: Twitter API integration
- `internal/processor/`: Main processing orchestration

## Migration from Node.js Version

1. Install Go 1.21+ and SOPS
2. Convert `.env` credentials to `secrets.yaml` format
3. Encrypt `secrets.yaml` with SOPS
4. Update any cron jobs or systemd services to use Go binary
5. Existing S3 archives are compatible with the Go version

## Troubleshooting

### Common Issues

**AWS Credential Errors:**
```bash
export AWS_PROFILE=your-profile
aws sso login  # If using SSO
```

**SOPS Decryption Failures:**
Ensure your KMS key permissions or PGP keys are properly configured

**Chrome/chromedp Issues:**
The application includes Chrome in Docker images, or uses system Chrome in standalone mode
