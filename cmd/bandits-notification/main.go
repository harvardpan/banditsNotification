package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bandits-notification/internal/config"
	"bandits-notification/internal/processor"
	"bandits-notification/internal/schedule"
	"bandits-notification/internal/scraper"
	"bandits-notification/internal/storage"
)

const (
	defaultConfigPath = "secrets.yaml"
)

func main() {
	// Parse command line flags
	var dryRun = flag.Bool("dry-run", false, "Generate local files instead of uploading to S3 or posting to Twitter")
	var noTweet = flag.Bool("no-tweet", false, "Upload to S3 but skip posting to Twitter (takes precedence over --dry-run)")
	flag.Parse()

	// Get config path from environment or use default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	s3Client, err := storage.NewS3Client(&cfg.AWS)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Verify S3 access by testing credentials
	log.Println("Verifying S3 access...")
	_, err = s3Client.FileExists("__credential_test__")
	if err != nil {
		// Check for the specific NoCredentialProviders error
		if strings.Contains(err.Error(), "NoCredentialProviders") {
			log.Fatalf("FATAL: No valid AWS credentials found. Please check your AWS credentials configuration.\n"+
				"Error: %v\n\n"+
				"Solutions:\n"+
				"1. Set AWS_PROFILE environment variable (e.g., export AWS_PROFILE=developmentpoweruser)\n"+
				"2. Run 'aws sso login' if using AWS SSO\n"+
				"3. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables\n"+
				"4. Configure credentials in ~/.aws/credentials file", err)
		} else {
			log.Fatalf("Failed to verify S3 access: %v", err)
		}
	}
	if *noTweet {
		log.Println("✅ S3 access verified (no-tweet mode: will read from and write to S3)")
	} else if *dryRun {
		log.Println("✅ S3 access verified (dry-run mode: will read from S3 but not write)")
	} else {
		log.Println("✅ S3 access verified")
	}

	// Twitter clients will be created per URL as needed
	log.Printf("Loaded configuration for %d URLs", len(cfg.App.URLs))

	scrapeClient := scraper.New()
	defer scrapeClient.Close()

	// Set up timezone
	timezone, err := cfg.GetTimezone()
	if err != nil {
		log.Printf("Warning: Failed to load timezone %s, using UTC: %v", cfg.App.DisplayTimezone, err)
		timezone = time.UTC
	}

	// Run once and exit (scheduling handled by Lambda/EventBridge)
	if *noTweet {
		log.Println("🚫 Starting bandits notification service (NO-TWEET MODE - single execution)")
	} else if *dryRun {
		log.Println("🔧 Starting bandits notification service (DRY-RUN MODE - single execution)")
	} else {
		log.Println("Starting bandits notification service (single execution)")
	}

	if err := processScheduleCheck(cfg, s3Client, scrapeClient, timezone, *dryRun, *noTweet); err != nil {
		log.Fatalf("Error during schedule check: %v", err)
	}

	log.Println("Schedule check completed successfully")
}

func processScheduleCheck(
	cfg *config.Config,
	s3Client *storage.S3Client,
	scrapeClient *scraper.Scraper,
	timezone *time.Location,
	dryRun bool,
	noTweet bool,
) error {
	timestamp := time.Now().In(timezone)
	log.Printf("Starting schedule check at %s", timestamp.Format("Monday, January 2nd 2006, 3:04:05 PM"))

	// Process each configured URL
	for _, urlConfig := range cfg.App.URLs {
		if err := processURL(urlConfig, cfg, s3Client, scrapeClient, timestamp, dryRun, noTweet); err != nil {
			log.Printf("Error processing URL %s: %v", urlConfig.URL, err)
			continue
		}
	}

	return nil
}

func processURL(
	urlConfig config.URLConfig,
	_ *config.Config,
	s3Client *storage.S3Client,
	scrapeClient *scraper.Scraper,
	timestamp time.Time,
	dryRun bool,
	noTweet bool,
) error {
	// Determine processing mode
	var mode processor.ProcessMode
	if noTweet {
		mode = processor.ModeNoTweet
	} else if dryRun {
		mode = processor.ModeDryRun
	} else {
		mode = processor.ModeNormal
	}

	opts := processor.ProcessOptions{
		Mode:      mode,
		Timestamp: timestamp,
	}

	result := processor.ProcessURL(urlConfig, s3Client, scrapeClient, opts)

	if result.Error != "" {
		// Handle dry-run mode specially since it requires file operations
		if result.Error == "dry-run-mode" && result.ChangesFound {
			return handleDryRunMode(result, timestamp, urlConfig)
		}
		return fmt.Errorf("error processing URL %s: %s", urlConfig.URL, result.Error)
	}

	return nil
}

// handleDryRunMode handles the file operations for dry-run mode
func handleDryRunMode(result *processor.ProcessResult, timestamp time.Time, urlConfig config.URLConfig) error {
	outputDir := fmt.Sprintf("dry-run-output/%s", result.URLIdentifier)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create dry-run output directory: %w", err)
	}

	// Save screenshot locally (instead of uploading to S3)
	screenshotFile := filepath.Join(outputDir, schedule.GetTimestampedFilename("schedule-screenshot", "png"))
	if err := os.WriteFile(screenshotFile, result.Screenshot, 0644); err != nil {
		log.Printf("Warning: Failed to save screenshot locally: %v", err)
	} else {
		log.Printf("🔧 Dry-run: Saved screenshot to %s (would upload to S3)", screenshotFile)
	}

	// Create tweet text and HTML representation (instead of posting to Twitter)
	// Ensure timestamp is in Eastern Time
	easternTime, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback to UTC if timezone loading fails
		easternTime = time.UTC
	}
	etTimestamp := timestamp.In(easternTime)

	day := etTimestamp.Day()
	timestampStr := fmt.Sprintf("%s, %s %d%s %d, %s",
		etTimestamp.Format("Monday"),
		etTimestamp.Format("January"),
		day,
		schedule.GetOrdinalSuffix(day),
		etTimestamp.Year(),
		etTimestamp.Format("3:04:05 PM"))
	tweetText := fmt.Sprintf("Latest Bandits Schedule as of %s. %s", timestampStr, urlConfig.URL)

	// Compare schedules to get diff info for HTML
	diff := schedule.CompareSchedules(result.PreviousSchedule, result.CurrentSchedule)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Tweet Preview - %s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
        .tweet { border: 1px solid #e1e8ed; border-radius: 16px; padding: 20px; background: white; margin: 20px 0; }
        .tweet-text { font-size: 20px; line-height: 1.3; margin-bottom: 12px; }
        .tweet-image { max-width: 100%%; border-radius: 12px; border: 1px solid #e1e8ed; }
        .tweet-meta { color: #657786; font-size: 14px; margin-top: 12px; }
        .timestamp { color: #1da1f2; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🔧 Dry-Run Mode: Tweet Preview</h1>
    <div class="tweet">
        <div class="tweet-text">%s</div>
        <img src="%s" alt="Schedule Screenshot" class="tweet-image" />
        <div class="tweet-meta">
            <span class="timestamp">%s</span> • 
            Would be posted to Twitter account: <strong>@%s</strong>
        </div>
    </div>
    <p><em>This tweet was NOT actually posted because dry-run mode is enabled.</em></p>
    <h2>📊 Schedule Changes Detected</h2>
    <ul>
        <li><strong>Added:</strong> %d entries</li>
        <li><strong>Modified:</strong> %d entries</li>
        <li><strong>Deleted:</strong> %d entries</li>
    </ul>
</body>
</html>`,
		urlConfig.Twitter.UserHandle,
		tweetText,
		filepath.Base(screenshotFile),
		timestampStr,
		urlConfig.Twitter.UserHandle,
		len(diff.Added), len(diff.Modified), len(diff.Deleted))

	htmlFile := filepath.Join(outputDir, schedule.GetTimestampedFilename("tweet-preview", "html"))
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		log.Printf("Warning: Failed to save tweet HTML preview: %v", err)
	} else {
		log.Printf("🔧 Dry-run: Saved tweet preview to %s (would post to Twitter)", htmlFile)
	}

	// Save current schedule data locally for inspection
	scheduleData, err := schedule.SerializeSchedule(result.CurrentSchedule)
	if err != nil {
		log.Printf("Warning: Failed to serialize current schedule: %v", err)
	} else {
		scheduleFile := filepath.Join(outputDir, schedule.GetTimestampedFilename("schedule", "json"))
		if err := os.WriteFile(scheduleFile, scheduleData, 0644); err != nil {
			log.Printf("Warning: Failed to save schedule data locally: %v", err)
		} else {
			log.Printf("🔧 Dry-run: Saved schedule data to %s (would save to S3)", scheduleFile)
		}
	}

	log.Printf("🔧 Dry-run: Files saved to %s (S3 writes and Twitter post skipped)", outputDir)
	return nil
}
