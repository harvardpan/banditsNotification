package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"bandits-notification/internal/config"
	"bandits-notification/internal/processor"
	"bandits-notification/internal/scraper"
	"bandits-notification/internal/storage"
	"github.com/aws/aws-lambda-go/lambda"
)

// Event represents the input event for the Lambda function
type Event struct {
	Source     string `json:"source"`
	DetailType string `json:"detail-type"`
	Detail     any    `json:"detail"`
}

// Response represents the response from the Lambda function
type Response struct {
	StatusCode    int         `json:"statusCode"`
	Message       string      `json:"message"`
	ProcessedURLs []URLResult `json:"processedUrls"`
}

// URLResult represents the result of processing a single URL
type URLResult struct {
	URL           string `json:"url"`
	TwitterHandle string `json:"twitterHandle"`
	ChangesFound  bool   `json:"changesFound"`
	TweetID       string `json:"tweetId,omitempty"`
	Error         string `json:"error,omitempty"`
}

func handleRequest(ctx context.Context, event Event) (Response, error) {
	log.Printf("Processing Lambda event: %+v", event)

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/tmp/secrets.yaml" // Default path in Lambda
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return Response{
			StatusCode: 500,
			Message:    fmt.Sprintf("Failed to load config: %v", err),
		}, err
	}

	// Initialize S3 client
	s3Client, err := storage.NewS3Client(&cfg.AWS)
	if err != nil {
		return Response{
			StatusCode: 500,
			Message:    fmt.Sprintf("Failed to create S3 client: %v", err),
		}, err
	}

	// Initialize scraper
	scraperClient := scraper.New()
	defer scraperClient.Close()

	var results []URLResult

	// Process each URL
	for _, urlConfig := range cfg.App.URLs {
		result := processURL(ctx, urlConfig, s3Client, scraperClient)
		results = append(results, result)

		// Log the result
		if result.Error != "" {
			log.Printf("Error processing %s: %s", result.URL, result.Error)
		} else if result.ChangesFound {
			log.Printf("Changes found for %s, posted tweet: %s", result.URL, result.TweetID)
		} else {
			log.Printf("No changes found for %s", result.URL)
		}
	}

	// Count successful and failed processing
	successful := 0
	failed := 0
	tweetsPosted := 0

	for _, result := range results {
		if result.Error != "" {
			failed++
		} else {
			successful++
			if result.ChangesFound {
				tweetsPosted++
			}
		}
	}

	return Response{
		StatusCode:    200,
		Message:       fmt.Sprintf("Processed %d URLs: %d successful, %d failed, %d tweets posted", len(results), successful, failed, tweetsPosted),
		ProcessedURLs: results,
	}, nil
}

func processURL(_ context.Context, urlConfig config.URLConfig, s3Client *storage.S3Client, scraperClient *scraper.Scraper) URLResult {
	opts := processor.ProcessOptions{
		Mode:      processor.ModeNormal,
		Timestamp: time.Now(),
	}

	result := processor.ProcessURL(urlConfig, s3Client, scraperClient, opts)

	// Convert processor result to lambda URLResult
	return URLResult{
		URL:           result.URL,
		TwitterHandle: result.TwitterHandle,
		ChangesFound:  result.ChangesFound,
		TweetID:       result.TweetID,
		Error:         result.Error,
	}
}

func main() {
	lambda.Start(handleRequest)
}
