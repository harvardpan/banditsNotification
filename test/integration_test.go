package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"bandits-notification/internal/config"
	"bandits-notification/internal/schedule"
	"bandits-notification/internal/scraper"
	"bandits-notification/internal/storage"
	"bandits-notification/internal/twitter"
)

// Full end-to-end integration test that actually scrapes URLs and tweets differences
func TestCompleteWorkflow_RealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if !shouldRunIntegrationTests() {
		t.Skip("Integration tests disabled. Set RUN_INTEGRATION_TESTS=true and provide test config to run.")
	}

	// Load test configuration
	cfg, err := config.LoadConfig("../test_config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	if len(cfg.App.URLs) == 0 {
		t.Fatalf("No URLs configured in test config")
	}

	// Initialize components
	s3Client, err := storage.NewS3Client(&cfg.AWS)
	if err != nil {
		t.Fatalf("Failed to create S3 client: %v", err)
	}

	scrapeClient := scraper.New()
	defer scrapeClient.Close()

	// Use first URL config for testing
	urlConfig := cfg.App.URLs[0]
	twitterClient := twitter.New(&urlConfig.Twitter)

	// Verify Twitter credentials
	user, err := twitterClient.VerifyCredentials()
	if err != nil {
		t.Fatalf("Failed to verify Twitter credentials: %v", err)
	}
	t.Logf("Twitter credentials verified for @%s", user.ScreenName)

	// Generate S3 keys based on URL identifier (test config gets -test suffix)
	isTestConfig := true // This is the test integration run
	urlIdentifier := schedule.GetURLIdentifier(urlConfig.URL, isTestConfig)
	previousScheduleKey := urlIdentifier + "/previousSchedule.json"
	archiveDir := urlIdentifier + "/archive/"

	var currentSchedule schedule.Schedule
	var screenshotData []byte
	var htmlData string

	t.Run("1. Scrape Target URL", func(t *testing.T) {
		t.Logf("Scraping URL: %s", urlConfig.URL)

		result, err := scrapeClient.ScrapePage(urlConfig.URL)
		if err != nil {
			t.Fatalf("Failed to scrape page: %v", err)
		}

		if result.HTML == "" {
			t.Fatal("Scraping returned empty HTML")
		}

		if len(result.Screenshot) == 0 {
			t.Fatal("Scraping returned empty screenshot")
		}

		screenshotData = result.Screenshot
		htmlData = result.HTML

		t.Logf("Successfully scraped: %d bytes HTML, %d bytes screenshot",
			len(result.HTML), len(result.Screenshot))
	})

	t.Run("2. Parse Schedule from HTML", func(t *testing.T) {
		parsedSchedule, err := schedule.ParseSchedule(htmlData)
		if err != nil {
			t.Fatalf("Failed to parse schedule: %v", err)
		}

		currentSchedule = parsedSchedule
		t.Logf("Successfully parsed schedule: %d entries", len(currentSchedule))

		// Log sample entries for debugging
		count := 0
		for key, entry := range currentSchedule {
			if count >= 3 { // Only log first 3 entries
				break
			}
			t.Logf("Schedule entry: %s -> %s at %s", key, entry.Location, entry.TimeBlock)
			count++
		}
	})

	var scheduleChanged bool
	var scheduleDiff schedule.ScheduleDiff
	var screenshotKey, archiveScheduleKey string // Store keys for cleanup

	t.Run("3. Compare with Previous Schedule", func(t *testing.T) {
		// Try to load previous schedule from S3
		previousSchedule, err := schedule.LoadSchedule(s3Client, urlIdentifier, "previousSchedule.json")
		if err != nil {
			t.Logf("No previous schedule found or failed to load: %v", err)
			// Treat as first run - everything is "added"
			previousSchedule = make(schedule.Schedule)
		} else {
			t.Logf("Loaded previous schedule: %d entries", len(previousSchedule))
		}

		// Compare schedules
		diffPtr := schedule.CompareSchedules(previousSchedule, currentSchedule)
		scheduleDiff = *diffPtr
		scheduleChanged = scheduleDiff.HasChanges()

		t.Logf("Schedule comparison: %d added, %d modified, %d deleted (HasChanges: %v)",
			len(scheduleDiff.Added), len(scheduleDiff.Modified), len(scheduleDiff.Deleted), scheduleChanged)

		// Log some example differences
		if len(scheduleDiff.Added) > 0 {
			t.Logf("Sample added entries:")
			count := 0
			for key, entry := range scheduleDiff.Added {
				if count >= 2 {
					break
				}
				t.Logf("  + %s: %s at %s", key, entry.Location, entry.TimeBlock)
				count++
			}
		}

		if len(scheduleDiff.Modified) > 0 {
			t.Logf("Sample modified entries:")
			count := 0
			for key, entry := range scheduleDiff.Modified {
				if count >= 2 {
					break
				}
				t.Logf("  ~ %s: %s at %s", key, entry.Location, entry.TimeBlock)
				count++
			}
		}

		if len(scheduleDiff.Deleted) > 0 {
			t.Logf("Sample deleted entries:")
			count := 0
			for key, entry := range scheduleDiff.Deleted {
				if count >= 2 {
					break
				}
				t.Logf("  - %s: %s at %s", key, entry.Location, entry.TimeBlock)
				count++
			}
		}
	})

	var tweetID string

	t.Run("4. Save Archives and Tweet if Changes Detected", func(t *testing.T) {
		if !scheduleChanged {
			t.Log("No schedule changes detected - skipping Twitter posting")
			return
		}

		t.Log("Schedule changes detected - proceeding with archival and tweeting")

		// Generate timestamped filenames
		timestamp := time.Now()
		screenshotFilename := timestamp.Format("2006-01-02_15-04-05") + "_schedule-screenshot.png"
		scheduleFilename := timestamp.Format("2006-01-02_15-04-05") + "_schedule.json"

		// Try S3 operations, but don't fail the test if credentials are missing
		s3Available := true

		// Save screenshot to S3 archive
		screenshotKey = archiveDir + screenshotFilename
		err = s3Client.UploadFile(screenshotData, screenshotKey)
		if err != nil {
			t.Logf("Warning: Failed to upload screenshot to S3 (possibly missing credentials): %v", err)
			s3Available = false
		} else {
			t.Logf("Successfully uploaded screenshot to S3: %s", screenshotKey)
		}

		// Save current schedule as previous schedule
		if s3Available {
			err = schedule.SaveSchedule(s3Client, currentSchedule, urlIdentifier, "previousSchedule.json")
			if err != nil {
				t.Logf("Warning: Failed to save current schedule as previous: %v", err)
				s3Available = false
			} else {
				t.Logf("Successfully saved current schedule as previous: %s", previousScheduleKey)
			}
		}

		// Save current schedule to archive
		if s3Available {
			archiveScheduleKey = archiveDir + scheduleFilename
			err = schedule.SaveSchedule(s3Client, currentSchedule, urlIdentifier, "archive/"+scheduleFilename)
			if err != nil {
				t.Logf("Warning: Failed to save schedule to archive: %v", err)
			} else {
				t.Logf("Successfully saved schedule to archive: %s", archiveScheduleKey)
			}
		}

		// Upload media and post tweet (need tweet ID for cleanup)
		mediaID, err := twitterClient.UploadMedia(screenshotData)
		if err != nil {
			t.Fatalf("Failed to upload media for tweet: %v", err)
		}

		// Create proper tweet message (matching the TweetWithImage format) with Eastern Time
		easternTime, err := time.LoadLocation("America/New_York")
		if err != nil {
			easternTime = time.UTC // Fallback
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
		tweetText := "🧪 INTEGRATION TEST - Latest Bandits Schedule as of " + timestampStr + ". " + urlConfig.URL

		tweetID, err = twitterClient.PostTweetWithMediaAndReturnID(tweetText, []string{mediaID})
		if err != nil {
			t.Fatalf("Failed to post tweet with media: %v", err)
		}
		t.Logf("Successfully posted tweet with ID: %s", tweetID)
	})

	t.Run("5. Verify Complete Workflow", func(t *testing.T) {
		if !scheduleChanged {
			t.Log("✅ Workflow complete - no changes detected, no tweet posted")
		} else {
			t.Log("✅ Workflow complete - changes detected, tweet posted")
			if tweetID != "" {
				t.Logf("✅ Successfully posted tweet with ID: %s", tweetID)
			}
		}

		// Try to verify that current schedule was saved (but don't fail if S3 unavailable)
		savedSchedule, err := schedule.LoadSchedule(s3Client, urlIdentifier, "previousSchedule.json")
		if err != nil {
			t.Logf("Could not verify saved schedule (possibly missing S3 credentials): %v", err)
		} else if len(savedSchedule) != len(currentSchedule) {
			t.Errorf("Saved schedule has %d entries, expected %d", len(savedSchedule), len(currentSchedule))
		} else {
			t.Logf("✅ Schedule successfully saved with %d entries", len(savedSchedule))
		}

		t.Logf("✅ Full end-to-end integration test completed successfully")
	})

	// If changes were detected and saved, run the workflow again to verify no differences are detected
	if scheduleChanged {
		t.Run("6. Repeat Workflow to Verify No Differences", func(t *testing.T) {
			t.Log("🔄 Running workflow again to confirm no differences are detected...")

			// Scrape the same URL again
			t.Log("Re-scraping URL...")
			result, err := scrapeClient.ScrapePage(urlConfig.URL)
			if err != nil {
				t.Fatalf("Failed to re-scrape page: %v", err)
			}

			// Parse schedule again
			t.Log("Re-parsing schedule...")
			newCurrentSchedule, err := schedule.ParseSchedule(result.HTML)
			if err != nil {
				t.Fatalf("Failed to re-parse schedule: %v", err)
			}
			t.Logf("Re-parsed schedule: %d entries", len(newCurrentSchedule))

			// Load the previous schedule (which should now be the saved currentSchedule)
			t.Log("Loading previously saved schedule...")
			previousSchedule, err := schedule.LoadSchedule(s3Client, urlIdentifier, "previousSchedule.json")
			if err != nil {
				t.Logf("Failed to load previously saved schedule (possibly missing S3 credentials): %v", err)
				t.Skip("Cannot verify repeat workflow without S3 access - skipping repeat test")
				return
			}
			t.Logf("Loaded previous schedule: %d entries", len(previousSchedule))

			// Compare schedules again
			t.Log("Comparing schedules...")
			repeatDiff := schedule.CompareSchedules(previousSchedule, newCurrentSchedule)
			repeatChanged := repeatDiff.HasChanges()

			t.Logf("Second comparison: %d added, %d modified, %d deleted (HasChanges: %v)",
				len(repeatDiff.Added), len(repeatDiff.Modified), len(repeatDiff.Deleted), repeatChanged)

			// This time we expect NO changes
			if repeatChanged {
				t.Errorf("❌ Expected no changes on second run, but detected: %d added, %d modified, %d deleted",
					len(repeatDiff.Added), len(repeatDiff.Modified), len(repeatDiff.Deleted))

				// Log details about unexpected changes
				if len(repeatDiff.Added) > 0 {
					t.Logf("Unexpected added entries:")
					for key, entry := range repeatDiff.Added {
						t.Logf("  + %s: %s at %s", key, entry.Location, entry.TimeBlock)
					}
				}
				if len(repeatDiff.Modified) > 0 {
					t.Logf("Unexpected modified entries:")
					for key, entry := range repeatDiff.Modified {
						t.Logf("  ~ %s: %s at %s", key, entry.Location, entry.TimeBlock)
					}
				}
				if len(repeatDiff.Deleted) > 0 {
					t.Logf("Unexpected deleted entries:")
					for key, entry := range repeatDiff.Deleted {
						t.Logf("  - %s: %s at %s", key, entry.Location, entry.TimeBlock)
					}
				}
			} else {
				t.Log("✅ Second run correctly detected no changes")
			}

			t.Log("✅ Repeat workflow verification completed")
		})
	}

	// Cleanup: Delete the test tweet and S3 files if created
	t.Cleanup(func() {
		// Delete test tweet if posted
		if tweetID != "" && scheduleChanged {
			err := twitterClient.DeleteTweet(tweetID)
			if err != nil {
				t.Logf("Warning: Failed to delete test tweet %s: %v", tweetID, err)
			} else {
				t.Logf("Successfully deleted test tweet: %s", tweetID)
			}
		}

		// Delete S3 files if created
		if scheduleChanged {
			// Delete the previous schedule file
			err := s3Client.DeleteFile(previousScheduleKey)
			if err != nil {
				t.Logf("Warning: Failed to delete previous schedule from S3 %s: %v", previousScheduleKey, err)
			} else {
				t.Logf("Successfully deleted previous schedule from S3: %s", previousScheduleKey)
			}

			// Delete archived files (screenshot and schedule) if they were created
			if screenshotKey != "" {
				err = s3Client.DeleteFile(screenshotKey)
				if err != nil {
					t.Logf("Warning: Failed to delete screenshot from S3 %s: %v", screenshotKey, err)
				} else {
					t.Logf("Successfully deleted screenshot from S3: %s", screenshotKey)
				}
			}

			if archiveScheduleKey != "" {
				err = s3Client.DeleteFile(archiveScheduleKey)
				if err != nil {
					t.Logf("Warning: Failed to delete archived schedule from S3 %s: %v", archiveScheduleKey, err)
				} else {
					t.Logf("Successfully deleted archived schedule from S3: %s", archiveScheduleKey)
				}
			}
		}
	})
}

// Helper function to check if integration tests should run
func shouldRunIntegrationTests() bool {
	return os.Getenv("RUN_INTEGRATION_TESTS") == "true"
}

// Test the configuration loading
func TestLoadConfiguration(t *testing.T) {
	// Skip in short mode (CI environment)
	if testing.Short() {
		t.Skip("Skipping configuration test in short mode")
	}

	// Test with the example config file
	if _, err := os.Stat("../test_config.yaml"); os.IsNotExist(err) {
		t.Skip("test_config.yaml not found - copy from test_config.yaml.example and configure")
	}

	cfg, err := config.LoadConfig("../test_config.yaml")
	if err != nil {
		// If it's a SOPS decryption error in CI, skip the test
		if strings.Contains(err.Error(), "Error getting data key") || strings.Contains(err.Error(), "sops") {
			t.Skipf("Skipping test due to SOPS decryption issue (likely in CI): %v", err)
		}
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Verify configuration structure
	if len(cfg.App.URLs) == 0 {
		t.Error("No URLs configured")
	} else if cfg.App.URLs[0].Twitter.UserHandle == "" {
		t.Error("Twitter user handle not configured for first URL")
	}

	if cfg.AWS.Region == "" {
		t.Error("AWS region not configured")
	}

	if len(cfg.App.URLs) == 0 {
		t.Error("No URLs configured for testing")
	}

	if len(cfg.App.URLs) > 0 {
		t.Logf("Successfully loaded test configuration for @%s", cfg.App.URLs[0].Twitter.UserHandle)
	}
}

// Error handling tests
func TestErrorHandling(t *testing.T) {
	t.Run("Invalid Config Path", func(t *testing.T) {
		_, err := config.LoadConfig("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for nonexistent config file")
		}
	})

	t.Run("Invalid S3 Config", func(t *testing.T) {
		cfg := &config.AWSConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
			Region:          "",
			S3Bucket:        "",
		}

		client, err := storage.NewS3Client(cfg)
		// Should create client even with empty config (AWS SDK handles this)
		if err != nil {
			t.Errorf("Unexpected error creating S3 client: %v", err)
		}
		if client == nil {
			t.Error("Expected non-nil S3 client")
		}
	})

	t.Run("Invalid Schedule Text", func(t *testing.T) {
		// Test with malformed schedule text
		malformedText := "This is not a valid schedule format"

		result, err := schedule.ParseSchedule(malformedText)
		if err != nil {
			t.Errorf("ParseSchedule should not error on malformed text: %v", err)
		}

		// Should return empty schedule, not error
		if len(result) != 0 {
			t.Errorf("Expected empty schedule for malformed text, got %d entries", len(result))
		}
	})
}
