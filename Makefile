# Makefile for bandits-notification Go application

.PHONY: build clean run test test-unit test-integration deps help lint encrypt decrypt encrypt-test decrypt-test docker-build docker-run lambda-deploy lambda-local lambda-invoke

# Variables
BINARY_NAME=bandits-notification
BUILD_DIR=bin
MAIN_PATH=./cmd/bandits-notification
KMS_KEY_ARN=arn:aws:kms:us-east-1:028036396420:alias/BanditsNotifierKMSKey

# Default target
help:
	@echo "Available targets:"
	@echo "  deps            - Download and tidy Go dependencies"
	@echo "  build           - Build the application binary"
	@echo "  run             - Run the application directly with go run"
	@echo "  run-dry         - Run the application in dry-run mode (no S3 writes/tweets)"
	@echo "  run-no-tweet    - Run the application without tweeting (writes to S3)"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests only (fast)"
	@echo "  test-integration - Run integration tests (requires config)"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run linting and formatting"
	@echo "  clean           - Remove built binaries"
	@echo "  encrypt         - Encrypt secrets.yaml with SOPS using KMS key"
	@echo "  decrypt         - Decrypt secrets.yaml for editing"
	@echo "  encrypt-test    - Encrypt test_config.yaml with SOPS using KMS key"
	@echo "  decrypt-test    - Decrypt test_config.yaml for editing"
	@echo "  docker-build    - Build Docker image (regular single execution)"
	@echo "  docker-build-lambda - Build Docker image for Lambda"
	@echo "  docker-run      - Run Docker container locally (single execution)"
	@echo "  docker-run-interactive - Run Docker container interactively"
	@echo "  lambda-deploy   - Deploy to AWS Lambda using infrastructure script"
	@echo "  lambda-local    - Run Lambda function locally with RIE"
	@echo "  lambda-invoke   - Invoke the deployed Lambda function in AWS"

# Download dependencies
deps:
	@echo "Downloading Go dependencies..."
	go mod tidy
	go mod download

# Build the application
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run: deps
	@echo "Running $(BINARY_NAME)..."
	AWS_SDK_LOAD_CONFIG=1 go run $(MAIN_PATH)

# Run the application in dry-run mode
run-dry: deps
	@echo "Running $(BINARY_NAME) in dry-run mode..."
	AWS_SDK_LOAD_CONFIG=1 go run $(MAIN_PATH) --dry-run

# Run the application in no-tweet mode
run-no-tweet: deps
	@echo "Running $(BINARY_NAME) in no-tweet mode..."
	AWS_SDK_LOAD_CONFIG=1 go run $(MAIN_PATH) --no-tweet

# Run all tests
test: test-unit test-integration

# Run unit tests only (fast)
test-unit: deps
	@echo "Running unit tests..."
	AWS_SDK_LOAD_CONFIG=1 go test -v -short ./...

# Run integration tests (requires configuration)
test-integration: deps
	@echo "Running integration tests..."
	@echo "Note: Set RUN_INTEGRATION_TESTS=true and configure test_config.yaml to enable"
	AWS_SDK_LOAD_CONFIG=1 AWS_PROFILE=developmentpoweruser RUN_INTEGRATION_TESTS=true go test -v ./test/...

# Run tests with coverage
test-coverage: deps
	@echo "Running tests with coverage..."
	AWS_SDK_LOAD_CONFIG=1 go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint and format code
lint: deps
	@echo "Running linting and formatting..."
	go fmt ./...
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run --timeout=5m; \
	elif test -f $(HOME)/go/bin/golangci-lint; then \
		echo "Running golangci-lint from ~/go/bin/..."; \
		$(HOME)/go/bin/golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found, install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean built artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Encrypt secrets with SOPS (requires SOPS to be configured)
encrypt:
	@echo "Encrypting secrets.yaml with SOPS using KMS key..."
	@if [ ! -f secrets.yaml ]; then echo "secrets.yaml not found!"; exit 1; fi
	sops -e --kms $(KMS_KEY_ARN) -i secrets.yaml
	@echo "secrets.yaml encrypted with KMS key: $(KMS_KEY_ARN)"

# Decrypt secrets for editing (requires SOPS to be configured)  
decrypt:
	@echo "Decrypting secrets.yaml for editing..."
	@if [ ! -f secrets.yaml ]; then echo "secrets.yaml not found!"; exit 1; fi
	sops -d -i secrets.yaml
	@echo "secrets.yaml decrypted - remember to encrypt again after editing"

# Encrypt test config with SOPS
encrypt-test:
	@echo "Encrypting test_config.yaml with SOPS using KMS key..."
	@if [ ! -f test_config.yaml ]; then echo "test_config.yaml not found!"; exit 1; fi
	sops -e --kms $(KMS_KEY_ARN) -i test_config.yaml
	@echo "test_config.yaml encrypted with KMS key: $(KMS_KEY_ARN)"

# Decrypt test config for editing
decrypt-test:
	@echo "Decrypting test_config.yaml for editing..."
	@if [ ! -f test_config.yaml ]; then echo "test_config.yaml not found!"; exit 1; fi
	sops -d -i test_config.yaml
	@echo "test_config.yaml decrypted - remember to encrypt again after editing"

# Install dependencies and build
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/"
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development setup complete"

# Quick smoke test
smoke-test: build
	@echo "Running smoke test..."
	@echo "Checking if binary runs without crashing..."
	AWS_SDK_LOAD_CONFIG=1 timeout 5s $(BUILD_DIR)/$(BINARY_NAME) || [ $$? -eq 124 ]
	@echo "Smoke test passed"

# Docker targets
docker-build:
	@echo "Building Docker image (regular)..."
	docker buildx build --platform linux/amd64 --provenance=false -t $(BINARY_NAME):latest .
	@echo "Docker image built: $(BINARY_NAME):latest"

docker-build-lambda:
	@echo "Building Docker image for Lambda..."
	docker buildx build --platform linux/amd64 --provenance=false -f Dockerfile.lambda -t $(BINARY_NAME):lambda .
	@echo "Lambda Docker image built: $(BINARY_NAME):lambda"

docker-run:
	@echo "Running Docker container..."
	docker run --rm \
		-v $(PWD)/secrets.yaml:/tmp/secrets.yaml:ro \
		-v ~/.aws:/root/.aws \
		-e AWS_REGION=us-east-1 \
		-e AWS_SDK_LOAD_CONFIG=1 \
		-e CONFIG_PATH=/tmp/secrets.yaml \
		-e AWS_PROFILE=developmentpoweruser \
		$(BINARY_NAME):latest

docker-run-interactive:
	@echo "Running Docker container interactively..."
	docker run --rm -it \
		-v $(PWD)/secrets.yaml:/tmp/secrets.yaml:ro \
		-v ~/.aws:/root/.aws \
		-e AWS_REGION=us-east-1 \
		-e AWS_SDK_LOAD_CONFIG=1 \
		-e CONFIG_PATH=/tmp/secrets.yaml \
		-e AWS_PROFILE=developmentpoweruser \
		$(BINARY_NAME):latest

# Lambda deployment
lambda-deploy:
	@echo "Deploying to AWS Lambda..."
	./infrastructure/deploy.sh

# Local Lambda testing with Runtime Interface Emulator
lambda-local: docker-build-lambda
	@echo "Starting Lambda function locally with RIE..."
	@echo "Function will be available at http://localhost:9000/2015-03-31/functions/function/invocations"
	docker-compose up --build lambda-rie

# Invoke the deployed Lambda function in AWS
lambda-invoke:
	@echo "Invoking Lambda function in AWS..."
	AWS_PROFILE=developmentpoweruser aws lambda invoke \
		--function-name bandits-notification-v3-bandits-notification \
		--payload '{"source":"manual","detail":{"manual_trigger":true}}' \
		--cli-binary-format raw-in-base64-out \
		/tmp/lambda-response.json
	@echo "Lambda response:"
	@cat /tmp/lambda-response.json
	@echo ""