package processor

import (
	"fmt"
	"log"
	"time"

	"bandits-notification/internal/config"
	"bandits-notification/internal/schedule"
	"bandits-notification/internal/scraper"
	"bandits-notification/internal/storage"
	"bandits-notification/internal/twitter"
)

// ProcessMode represents different execution modes
type ProcessMode int

const (
	ModeNormal ProcessMode = iota
	ModeDryRun
	ModeNoTweet
)

// ProcessOptions contains options for processing URLs
type ProcessOptions struct {
	Mode      ProcessMode
	Timestamp time.Time
}

// ProcessResult contains the result of processing a URL
type ProcessResult struct {
	URL           string
	TwitterHandle string
	ChangesFound  bool
	TweetID       string
	Error         string

	// Internal data for further processing
	CurrentSchedule  schedule.Schedule
	PreviousSchedule schedule.Schedule
	Screenshot       []byte
	URLIdentifier    string
}

// ProcessURL handles the core logic of processing a single URL for schedule changes
func ProcessURL(urlConfig config.URLConfig, s3Client *storage.S3Client, scrapeClient *scraper.Scraper, opts ProcessOptions) *ProcessResult {
	result := &ProcessResult{
		URL:           urlConfig.URL,
		TwitterHandle: urlConfig.Twitter.UserHandle,
		ChangesFound:  false,
	}

	log.Printf("Processing URL: %s", urlConfig.URL)

	// Create Twitter client for this URL
	twitterClient := twitter.New(&urlConfig.Twitter)

	// Verify Twitter credentials
	user, err := twitterClient.VerifyCredentials()
	if err != nil {
		result.Error = fmt.Sprintf("failed to verify Twitter credentials for %s: %v", urlConfig.URL, err)
		return result
	}
	log.Printf("Connected to Twitter as @%s for URL %s", user.ScreenName, urlConfig.URL)

	// Scrape the webpage
	scrapeResult, err := scrapeClient.ScrapePage(urlConfig.URL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to scrape page: %v", err)
		return result
	}
	result.Screenshot = scrapeResult.Screenshot

	// Parse the schedule directly from HTML
	currentSchedule, err := schedule.ParseSchedule(scrapeResult.HTML)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse schedule: %v", err)
		return result
	}
	result.CurrentSchedule = currentSchedule

	// Generate S3 keys based on URL identifier
	isTestConfig := false // This is production mode (not test)
	urlIdentifier := schedule.GetURLIdentifier(urlConfig.URL, isTestConfig)
	result.URLIdentifier = urlIdentifier

	// Load previous schedule for comparison from S3
	previousSchedule, err := schedule.LoadSchedule(s3Client, urlIdentifier, "previousSchedule.json")
	if err != nil {
		log.Printf("Warning: Could not load previous schedule (this is normal for first run): %v", err)
		previousSchedule = make(schedule.Schedule)
	}
	result.PreviousSchedule = previousSchedule

	// Compare schedules
	diff := schedule.CompareSchedules(previousSchedule, currentSchedule)

	// Check if there are changes
	if !diff.HasChanges() {
		log.Println("No schedule changes detected")
		return result
	}

	result.ChangesFound = true
	log.Printf("Schedule changes detected! Added: %d, Modified: %d, Deleted: %d",
		len(diff.Added), len(diff.Modified), len(diff.Deleted))

	// Handle different processing modes
	switch opts.Mode {
	case ModeNoTweet:
		return handleNoTweetMode(result, s3Client, twitterClient, opts)
	case ModeDryRun:
		return handleDryRunMode(result, s3Client, twitterClient, opts)
	case ModeNormal:
		return handleNormalMode(result, s3Client, twitterClient, opts)
	default:
		result.Error = fmt.Sprintf("unknown processing mode: %v", opts.Mode)
		return result
	}
}

// handleNoTweetMode uploads to S3 but skips Twitter posting
func handleNoTweetMode(result *ProcessResult, s3Client *storage.S3Client, twitterClient *twitter.Client, opts ProcessOptions) *ProcessResult {
	log.Println("🚫 No-tweet mode: Uploading to S3 but skipping Twitter post")

	// Upload screenshot to S3 archive folder
	screenshotKey := schedule.GetTimestampedFilename("schedule-screenshot", "png")
	archiveDir := result.URLIdentifier + "/archive/"
	screenshotS3Key := archiveDir + screenshotKey
	if err := s3Client.UploadFile(result.Screenshot, screenshotS3Key); err != nil {
		log.Printf("Warning: Failed to upload screenshot to S3: %v", err)
	} else {
		log.Printf("✅ Uploaded screenshot to S3: %s", screenshotS3Key)
	}

	// Save current schedule as the new previous schedule
	if err := schedule.SaveSchedule(s3Client, result.CurrentSchedule, result.URLIdentifier, "previousSchedule.json"); err != nil {
		log.Printf("Warning: Failed to save current schedule: %v", err)
	} else {
		log.Printf("✅ Saved current schedule as previous")
	}

	// Archive the current schedule with timestamp
	archiveKey := schedule.GetTimestampedFilename("schedule", "json")
	if err := schedule.SaveSchedule(s3Client, result.CurrentSchedule, result.URLIdentifier, "archive/"+archiveKey); err != nil {
		log.Printf("Warning: Failed to archive schedule: %v", err)
	} else {
		log.Printf("✅ Archived schedule to S3")
	}

	log.Printf("🚫 No-tweet mode: S3 operations completed, Twitter post skipped")
	return result
}

// handleDryRunMode saves files locally and creates HTML representation
func handleDryRunMode(result *ProcessResult, s3Client *storage.S3Client, twitterClient *twitter.Client, opts ProcessOptions) *ProcessResult {
	// Dry-run mode: save files locally and create HTML representation
	// (but skip S3 uploads and Twitter posting)

	// Mark as dry-run mode requiring caller implementation
	result.Error = "dry-run-mode"
	return result
}

// handleNormalMode uploads to S3 and posts to Twitter
func handleNormalMode(result *ProcessResult, s3Client *storage.S3Client, twitterClient *twitter.Client, opts ProcessOptions) *ProcessResult {
	// Upload screenshot to S3 archive folder
	screenshotKey := schedule.GetTimestampedFilename("schedule-screenshot", "png")
	archiveDir := result.URLIdentifier + "/archive/"
	screenshotS3Key := archiveDir + screenshotKey
	if err := s3Client.UploadFile(result.Screenshot, screenshotS3Key); err != nil {
		log.Printf("Warning: Failed to upload screenshot to S3: %v", err)
	}

	// Tweet the screenshot using the correct pattern
	// Step 1: Upload media
	mediaID, err := twitterClient.UploadMedia(result.Screenshot)
	if err != nil {
		result.Error = fmt.Sprintf("failed to upload media: %v", err)
		return result
	}

	// Step 2: Create tweet text with Eastern Time
	easternTime, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback to UTC if timezone loading fails
		easternTime = time.UTC
	}
	etTimestamp := opts.Timestamp.In(easternTime)

	day := etTimestamp.Day()
	timestampStr := fmt.Sprintf("%s, %s %d%s %d, %s",
		etTimestamp.Format("Monday"),
		etTimestamp.Format("January"),
		day,
		schedule.GetOrdinalSuffix(day),
		etTimestamp.Year(),
		etTimestamp.Format("3:04:05 PM"))
	tweetText := fmt.Sprintf("Latest Bandits Schedule as of %s. %s", timestampStr, result.URL)

	// Step 3: Post tweet with media
	tweetID, err := twitterClient.PostTweetWithMediaAndReturnID(tweetText, []string{mediaID})
	if err != nil {
		result.Error = fmt.Sprintf("failed to post tweet: %v", err)
		return result
	}
	result.TweetID = tweetID
	log.Printf("Successfully posted tweet with schedule update! Tweet ID: %s", tweetID)

	// Save current schedule as the new previous schedule
	if err := schedule.SaveSchedule(s3Client, result.CurrentSchedule, result.URLIdentifier, "previousSchedule.json"); err != nil {
		log.Printf("Warning: Failed to save current schedule: %v", err)
	}

	// Archive the current schedule with timestamp
	archiveKey := schedule.GetTimestampedFilename("schedule", "json")
	if err := schedule.SaveSchedule(s3Client, result.CurrentSchedule, result.URLIdentifier, "archive/"+archiveKey); err != nil {
		log.Printf("Warning: Failed to archive schedule: %v", err)
	}

	return result
}
