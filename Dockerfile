# Multi-stage build starting from chromedp/headless-shell
FROM golang:1.25.1-alpine3.22 AS build

# Set the working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o /app/main ./cmd/bandits-notification

# Use chromedp/headless-shell as the base - it has Chrome pre-installed
FROM chromedp/headless-shell:latest

# Install necessary tools
USER root
RUN apt-get update && apt-get install -y \
    wget curl unzip ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install SOPS
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then SOPS_ARCH="amd64"; elif [ "$ARCH" = "aarch64" ]; then SOPS_ARCH="arm64"; else SOPS_ARCH="amd64"; fi && \
    wget "https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.$SOPS_ARCH" -O /usr/local/bin/sops && \
    chmod +x /usr/local/bin/sops

# Copy the built application
COPY --from=build /app/main /app/bandits-notification

# Set environment variables for Chrome
ENV CHROME_BIN=/usr/bin/google-chrome-stable
ENV DISPLAY=:99

# Run as root for Lambda compatibility
USER root

# Create writable directories for AWS SSO cache
RUN mkdir -p /root/.aws/sso/cache && chmod 755 /root/.aws/sso/cache

# Override the entrypoint from base image to run our app directly
ENTRYPOINT []
CMD [ "/app/bandits-notification" ]