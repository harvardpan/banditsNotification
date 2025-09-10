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

### Testing Strategy
- **Unit Tests**: Each internal package has comprehensive unit tests (`*_test.go` files)
- **Integration Tests**: End-to-end workflow tests in `test/integration_test.go`
- **Test Configuration**: Separate `test_config.yaml` for integration test credentials
- **Environment Isolation**: Integration tests controlled by `RUN_INTEGRATION_TESTS` environment variable
- **Continuous Testing**: Automated test execution in GitHub Actions CI pipeline

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

### CI/CD Pipeline (GitHub Actions)
- Automated deployment triggered on pushes to `main` or `develop` branches
- OIDC authentication for secure, keyless AWS access
- Multi-stage pipeline: test → lint → build → deploy
- ECR image builds and Lambda function updates
- Security scanning with Trivy on pull requests
- Integration test execution in staging environment

## Code Architecture

### Main Entry Points
- `cmd/bandits-notification/main.go` - Standalone application for direct execution or cron jobs
- `cmd/lambda/main.go` - AWS Lambda handler for serverless execution

### Core Internal Packages
- `internal/config/` - SOPS-encrypted configuration management with secrets.yaml support
- `internal/scraper/` - Web scraping using chromedp (headless Chrome)
- `internal/schedule/` - Schedule parsing, comparison, serialization, and time utilities
- `internal/storage/` - AWS S3 integration for archiving screenshots and schedule data
- `internal/twitter/` - Custom Twitter API v1 client with OAuth1 implementation for posting tweets with media
- `internal/processor/` - Main processing logic orchestrating scraping, comparison, and posting

### Key Dependencies
- **chromedp** - Headless Chrome automation for web scraping and screenshots
- **SOPS** - Encrypted secret management (replaces HashiCorp Vault)
- **aws-sdk-go** - AWS S3 file uploads and Lambda runtime
- **goquery** - HTML parsing for schedule extraction
- **gopkg.in/yaml.v3** - YAML configuration parsing
- Built-in **time** package - Date/time handling with timezone support

### Key Implementation Details
- **Custom Twitter OAuth1**: Complete OAuth1 implementation for Twitter API v1, including signature generation and request signing
- **Environment-Aware Scraper**: chromedp configuration automatically adapts between local development and Lambda execution environments
- **Dual Screenshot Strategy**: Captures both screenshots and corresponding HTML content for comprehensive archival

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

### Infrastructure as Code
- **CloudFormation Templates**: Complete AWS infrastructure defined in `infrastructure/` directory
  - `lambda-stack.yaml` - Lambda function, IAM roles, ECR repository, and EventBridge schedule
  - `github-actions-role.yaml` - OIDC provider and IAM role for GitHub Actions CI/CD
- **Automated Deployment**: Infrastructure updates deployed via GitHub Actions using CloudFormation
- **Environment Management**: Separate configurations for development and production environments

# Using Gemini CLI for Large Codebase Analysis

When analyzing large codebases or multiple files that might exceed context limits, use the Gemini CLI with its massive context window.

## File and Directory Inclusion Syntax

Use the `@` syntax to include files and directories in your Gemini prompts. The paths should be relative to WHERE you run the gemini command:

### Examples:

**Single file analysis:**
```bash
gemini "@src/main.py Explain this file's purpose and structure"
```

**Multiple files:**
```bash
gemini "@package.json @src/index.js Analyze the dependencies used in the code"
```

**Entire directory:**
```bash
gemini "@src/ Summarize the architecture of this codebase"
```

**Multiple directories:**
```bash
gemini "@src/ @tests/ Analyze test coverage for the source code"
```

**Current directory and subdirectories:**
```bash
gemini "@./ Give me an overview of this entire project"
```

## Implementation Verification Examples

**Check if a feature is implemented:**
```bash
gemini "@src/ @lib/ Has dark mode been implemented in this codebase? Show me the relevant files and functions"
```

**Verify authentication implementation:**
```bash
gemini "@src/ @middleware/ Is JWT authentication implemented? List all auth-related endpoints and middleware"
```

**Check for specific patterns:**
```bash
gemini "@src/ Are there any React hooks that handle WebSocket connections? List them with file paths"
```

**Verify error handling:**
```bash
gemini "@src/ @api/ Is proper error handling implemented for all API endpoints? Show examples of try-catch blocks"
```

**Check for rate limiting:**
```bash
gemini "@backend/ @middleware/ Is rate limiting implemented for the API? Show the implementation details"
```

**Verify caching strategy:**
```bash
gemini "@src/ @lib/ @services/ Is Redis caching implemented? List all cache-related functions and their usage"
```

**Check for specific security measures:**
```bash
gemini "@src/ @api/ Are SQL injection protections implemented? Show how user inputs are sanitized"
```

**Verify test coverage for features:**
```bash
gemini "@src/payment/ @tests/ Is the payment processing module fully tested? List all test cases"
```

## When to Use Gemini CLI

Use `gemini` when:
- Analyzing entire codebases or large directories
- Comparing multiple large files  
- Need to understand project-wide patterns or architecture
- Current context window is insufficient for the task
- Working with files totaling more than 100KB
- Verifying if specific features, patterns, or security measures are implemented
- Checking for the presence of certain coding patterns across the entire codebase

## Important Notes

- Paths in `@` syntax are relative to your current working directory when invoking gemini
- The CLI will include file contents directly in the context
- No need for `--yolo` flag for read-only analysis
- Gemini's context window can handle entire codebases that would overflow Claude's context
- When checking implementations, be specific about what you're looking for to get accurate results
  