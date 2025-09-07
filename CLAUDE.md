# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that monitors the Brookline Bandits 12U and 14U baseball schedule webpages and automatically tweets schedule changes with screenshots. The application:

- Scrapes web pages using chromedp for screenshots and content parsing
- Compares current schedule with archived versions to detect changes
- Posts tweets with screenshots when changes are detected
- Stores archived schedule data in JSON format and uploads to AWS S3
- Uses SOPS (Secrets OPerationS) for encrypted secret management
- Runs as single execution (scheduled externally via AWS Lambda/EventBridge)
- Supports multiple deployment modes: standalone binary, Docker container, or AWS Lambda

## Common Commands

### Development
- `make run` - Run the main application
- `make run-dry` - Run in dry-run mode (no S3 writes/tweets, saves files locally)
- `make run-no-tweet` - Run without tweeting (writes to S3 but no Twitter posts)
- `make build` - Build the application binary
- `make deps` - Download and tidy Go dependencies

### Testing
- `make test` - Run all tests (unit + integration)
- `make test-unit` - Run unit tests only (fast)
- `make test-integration` - Run integration tests (requires configuration)
- `make test-coverage` - Run tests with coverage report

### Linting
- `make lint` - Run linting and formatting (go fmt, go vet, golangci-lint)

### Secret Management
- `make encrypt` - Encrypt secrets.yaml with SOPS using AWS KMS
- `make decrypt` - Decrypt secrets.yaml for editing
- `make encrypt-test` / `make decrypt-test` - Same for test_config.yaml

### Docker & Lambda
- `make docker-build` - Build Docker image for standalone execution
- `make docker-build-lambda` - Build Docker image for AWS Lambda
- `make lambda-deploy` - Deploy to AWS Lambda using infrastructure scripts
- `make lambda-invoke` - Invoke the deployed Lambda function

## Code Architecture

### Main Entry Points
- `cmd/bandits-notification/main.go` - Standalone application for direct execution or cron jobs
- `cmd/lambda/main.go` - AWS Lambda handler for serverless execution

### Core Internal Packages
- `internal/config/` - SOPS-encrypted configuration management with secrets.yaml support
- `internal/scraper/` - Web scraping using chromedp (headless Chrome)
- `internal/schedule/` - Schedule parsing, comparison, serialization, and time utilities
- `internal/storage/` - AWS S3 integration for archiving screenshots and schedule data
- `internal/twitter/` - Twitter API v1 integration for posting tweets with media
- `internal/processor/` - Main processing logic orchestrating scraping, comparison, and posting

### Key Dependencies
- **chromedp** - Headless Chrome automation for web scraping and screenshots
- **SOPS** - Encrypted secret management (replaces HashiCorp Vault)
- **aws-sdk-go** - AWS S3 file uploads and Lambda runtime
- **goquery** - HTML parsing for schedule extraction
- **gopkg.in/yaml.v3** - YAML configuration parsing
- Built-in **time** package - Date/time handling with timezone support

### Data Flow
1. Application loads encrypted secrets from SOPS-managed secrets.yaml
2. chromedp scrapes the target webpage and takes screenshot
3. Schedule data is parsed from HTML and compared with archived version from S3
4. If changes detected, screenshot is uploaded to S3 and tweeted
5. New schedule data is archived to S3 for future comparisons

### Configuration
The application uses a SOPS-encrypted `secrets.yaml` file containing:
- Twitter API credentials (consumer key/secret, access token/secret, user handle)
- AWS credentials (optional - can use default credential chain) and S3 bucket settings
- Runtime configuration (URLs to monitor, timezone, etc.)

Multiple URLs can be configured for monitoring different schedules with different Twitter accounts.

### Processing Modes
- **Normal Mode** - Full operation: scrape, compare, upload to S3, post tweets
- **No-Tweet Mode** - Scrape, compare, upload to S3, but skip Twitter posts
- **Dry-Run Mode** - Scrape, compare, save files locally, but skip S3 uploads and Twitter posts

### Deployment Architecture
The Go implementation supports multiple deployment patterns:
- **Standalone Binary** - Direct execution or cron job scheduling
- **Docker Container** - Containerized execution in any container runtime
- **AWS Lambda** - Serverless execution triggered by EventBridge on a schedule