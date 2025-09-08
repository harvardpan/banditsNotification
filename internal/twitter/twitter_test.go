package twitter

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"bandits-notification/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.TwitterConfig{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		AccessToken:       "test-access-token",
		AccessTokenSecret: "test-access-token-secret",
		UserHandle:        "testuser",
	}

	client := New(cfg)

	if client == nil {
		t.Fatal("New() returned nil client")
	}

	if client.config != cfg {
		t.Error("New() did not store config correctly")
	}

	if client.httpClient == nil {
		t.Error("New() did not create HTTP client")
	}
}

func TestVerifyCredentials(t *testing.T) {
	cfg := &config.TwitterConfig{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		AccessToken:       "test-access-token",
		AccessTokenSecret: "test-access-token-secret",
		UserHandle:        "testuser",
	}

	client := New(cfg)

	// This should fail with real API call since we don't have valid credentials
	user, err := client.VerifyCredentials()

	// Should get an error for invalid credentials
	if err == nil {
		t.Error("VerifyCredentials() expected error for invalid credentials but got none")
	} else {
		t.Logf("VerifyCredentials() correctly failed with: %v", err)
	}

	// Should not return a user on error
	if user != nil {
		t.Error("VerifyCredentials() should return nil user on error")
	}
}

func TestUploadMediaAndPostTweet(t *testing.T) {
	cfg := &config.TwitterConfig{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		AccessToken:       "test-access-token",
		AccessTokenSecret: "test-access-token-secret",
		UserHandle:        "testuser",
	}

	client := New(cfg)

	tests := []struct {
		name      string
		message   string
		imageData []byte
		wantErr   bool
	}{
		{
			name:      "valid tweet with image",
			message:   "Test tweet with image",
			imageData: createTestPNG(),
			wantErr:   true, // Expect error with fake credentials
		},
		{
			name:      "text only tweet",
			message:   "Test text-only tweet",
			imageData: nil,
			wantErr:   true, // Expect error with fake credentials
		},
		{
			name:      "empty image data",
			message:   "Test with empty image",
			imageData: []byte{},
			wantErr:   true, // Expect error with fake credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mediaIDs []string
			var err error

			// Step 1: Upload media if image data is provided and not empty
			if len(tt.imageData) > 0 {
				mediaID, err := client.UploadMedia(tt.imageData)
				if tt.wantErr && err != nil {
					t.Logf("UploadMedia() correctly failed with: %v", err)
					return // Expected failure at upload stage
				}
				if !tt.wantErr && err != nil {
					t.Errorf("UploadMedia() unexpected error: %v", err)
					return
				}
				if mediaID != "" {
					mediaIDs = []string{mediaID}
				}
			}

			// Step 2: Post tweet with media
			tweetID, err := client.PostTweetWithMediaAndReturnID(tt.message, mediaIDs)

			if tt.wantErr {
				if err == nil {
					t.Error("PostTweetWithMediaAndReturnID() expected error for invalid credentials but got none")
				} else {
					t.Logf("PostTweetWithMediaAndReturnID() correctly failed with: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("PostTweetWithMediaAndReturnID() unexpected error: %v", err)
				}
				if tweetID == "" {
					t.Error("PostTweetWithMediaAndReturnID() returned empty tweet ID")
				}
			}
		})
	}
}

func TestOAuth1Signature(t *testing.T) {
	cfg := &config.TwitterConfig{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		AccessToken:       "test-access-token",
		AccessTokenSecret: "test-access-token-secret",
		UserHandle:        "testuser",
	}

	client := New(cfg)

	// Test OAuth1 signature generation
	oauthParams := map[string]string{
		"oauth_consumer_key":     "test-consumer-key",
		"oauth_token":            "test-access-token",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1234567890",
		"oauth_nonce":            "test-nonce",
		"oauth_version":          "1.0",
	}

	signature, err := client.generateOAuth1Signature("POST", "https://api.twitter.com/1.1/statuses/update.json", oauthParams, nil)
	if err != nil {
		t.Errorf("generateOAuth1Signature() error = %v", err)
	}

	if signature == "" {
		t.Error("generateOAuth1Signature() returned empty signature")
	}

	// Test that same inputs produce same signature
	signature2, err := client.generateOAuth1Signature("POST", "https://api.twitter.com/1.1/statuses/update.json", oauthParams, nil)
	if err != nil {
		t.Errorf("generateOAuth1Signature() error = %v", err)
	}

	if signature != signature2 {
		t.Error("generateOAuth1Signature() should return consistent results for same inputs")
	}
}

func TestGenerateNonce(t *testing.T) {
	// Test that generateNonce produces different values
	nonce1 := generateNonce()
	nonce2 := generateNonce()

	if nonce1 == "" {
		t.Error("generateNonce() returned empty string")
	}

	if nonce2 == "" {
		t.Error("generateNonce() returned empty string")
	}

	if nonce1 == nonce2 {
		t.Error("generateNonce() should return different values on subsequent calls")
	}

	// Test that nonce is base64 encoded (should not contain invalid characters)
	for _, char := range nonce1 {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '+' || char == '/' || char == '=') {
			t.Errorf("generateNonce() returned invalid base64 character: %c", char)
		}
	}
}

// Integration test that actually posts and deletes a tweet
func TestTwitterClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Try to load real config for integration testing
	configPath := "../../test_config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../../secrets.yaml"
	}

	// Try to load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Skipf("Skipping integration test - cannot load config: %v", err)
	}

	if len(cfg.App.URLs) == 0 {
		t.Skip("Skipping integration test - no URLs configured")
	}

	// Use first URL config for testing
	twitterClient := New(&cfg.App.URLs[0].Twitter)

	// Test credential verification
	user, err := twitterClient.VerifyCredentials()
	if err != nil {
		t.Skipf("Skipping integration test - invalid Twitter credentials: %v", err)
	}
	t.Logf("VerifyCredentials succeeded for user: @%s", user.ScreenName)

	// Create test image data using Go's image package (similar to Sharp)
	testImageData := createTestPNG()

	// Test posting a tweet with image
	testMessage := fmt.Sprintf("🧪 Integration test tweet - %s", time.Now().Format("2006-01-02 15:04:05"))

	// Track tweet ID for cleanup
	var tweetID string

	t.Run("Post Text-Only Tweet", func(t *testing.T) {
		// Post a text-only tweet first to test basic functionality
		tweetID, err = twitterClient.PostTweetWithMediaAndReturnID(testMessage, nil)
		if err != nil {
			t.Fatalf("Failed to post text-only tweet: %v", err)
		}
		t.Logf("Successfully posted text-only tweet: %s", tweetID)
	})

	// Test media upload with proper PNG
	t.Run("Media Upload Test", func(t *testing.T) {
		mediaID, err := twitterClient.UploadMedia(testImageData)
		if err != nil {
			t.Fatalf("Media upload failed: %v", err)
		}
		t.Logf("Media upload succeeded: %s", mediaID)

		// Test posting tweet with the uploaded media
		testMessageWithImage := fmt.Sprintf("🧪 Integration test with image - %s", time.Now().Format("2006-01-02 15:04:05"))
		imageTweetID, err := twitterClient.PostTweetWithMediaAndReturnID(testMessageWithImage, []string{mediaID})
		if err != nil {
			t.Fatalf("Failed to post tweet with image: %v", err)
		}
		t.Logf("Successfully posted tweet with image: %s", imageTweetID)

		// Clean up the image tweet
		t.Cleanup(func() {
			if imageTweetID != "" {
				err := twitterClient.DeleteTweet(imageTweetID)
				if err != nil {
					t.Logf("Warning: Failed to delete image tweet %s: %v", imageTweetID, err)
				} else {
					t.Logf("Successfully deleted image tweet: %s", imageTweetID)
				}
			}
		})
	})

	// Cleanup: Delete the tweet
	t.Cleanup(func() {
		if tweetID != "" {
			err := twitterClient.DeleteTweet(tweetID)
			if err != nil {
				t.Logf("Warning: Failed to delete test tweet %s: %v", tweetID, err)
			} else {
				t.Logf("Successfully deleted test tweet: %s", tweetID)
			}
		}
	})
}

// Performance tests
func BenchmarkGenerateNonce(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nonce := generateNonce()
		if nonce == "" {
			b.Error("generateNonce() returned empty string")
		}
	}
}

func BenchmarkOAuth1Signature(b *testing.B) {
	cfg := &config.TwitterConfig{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		AccessToken:       "test-access-token",
		AccessTokenSecret: "test-access-token-secret",
		UserHandle:        "testuser",
	}

	client := New(cfg)
	oauthParams := map[string]string{
		"oauth_consumer_key":     "test-consumer-key",
		"oauth_token":            "test-access-token",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1234567890",
		"oauth_nonce":            "test-nonce",
		"oauth_version":          "1.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.generateOAuth1Signature("POST", "https://api.twitter.com/1.1/statuses/update.json", oauthParams, nil)
		if err != nil {
			b.Errorf("generateOAuth1Signature() error = %v", err)
		}
	}
}

// Test helper functions
func TestTwitterConfig(t *testing.T) {
	cfg := &config.TwitterConfig{}

	// Test that empty config doesn't cause panics
	client := New(cfg)
	if client == nil {
		t.Error("New() returned nil with empty config")
	}

	// Test that client stores config correctly
	if client.config != cfg {
		t.Error("New() did not store config reference correctly")
	}

	// Test HTTP client is created
	if client.httpClient == nil {
		t.Error("New() did not create HTTP client")
	}
}

// createTestPNG creates a proper PNG image using Go's image package
// This mimics what Sharp or Puppeteer would create
func createTestPNG() []byte {
	// Create a 100x100 RGBA image with a blue background (similar to Sharp test)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Fill with blue color (similar to the Sharp config: r: 0, g: 100, b: 200)
	blue := color.RGBA{0, 100, 200, 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, blue)
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(fmt.Sprintf("Failed to create test PNG: %v", err))
	}

	return buf.Bytes()
}
