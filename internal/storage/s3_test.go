package storage

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"bandits-notification/internal/config"
)

// Test data
var testData = []byte("test schedule data")
var testKey = "test-schedule.json"

// Helper function to create test config from actual config file
func getTestConfig(t testing.TB) *config.AWSConfig {
	// Try to load the actual configuration
	configPath := "../../test_config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../../secrets.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Skipf("Skipping integration test - cannot load config: %v", err)
	}

	if cfg.AWS.S3Bucket == "" {
		t.Skip("Skipping integration test - no S3 bucket configured")
	}

	return &cfg.AWS
}

// Helper function to check if integration tests should run
func shouldRunIntegrationTests(t testing.TB) bool {
	// Check if we're running in short mode
	if testing.Short() {
		return false
	}

	// Try to load config to see if we have S3 credentials
	configPath := "../../test_config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../../secrets.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return false
	}

	return cfg.AWS.S3Bucket != ""
}

func TestNewS3Client(t *testing.T) {
	cfg := &config.AWSConfig{
		Region:   "us-east-1",
		S3Bucket: "test-bucket",
	}

	client, err := NewS3Client(cfg)
	if err != nil {
		t.Fatalf("NewS3Client() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewS3Client() returned nil client")
	}

	if client.bucket != "test-bucket" {
		t.Errorf("NewS3Client() bucket = %q, want %q", client.bucket, "test-bucket")
	}
}

func TestIsNoSuchKeyError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "NoSuchKey error",
			errMsg:   "NoSuchKey: The specified key does not exist.",
			expected: true,
		},
		{
			name:     "NotFound error",
			errMsg:   "NotFound: 404 page not found",
			expected: true,
		},
		{
			name:     "Other error",
			errMsg:   "AccessDenied: Access Denied",
			expected: false,
		},
		{
			name:     "nil error",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}
			
			result := isNoSuchKeyError(err)
			if result != tt.expected {
				t.Errorf("isNoSuchKeyError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test error type for mocking AWS errors
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			str:      "NoSuchKey: The specified key does not exist",
			substr:   "NoSuchKey",
			expected: true,
		},
		{
			name:     "does not contain substring",
			str:      "AccessDenied: Access Denied",
			substr:   "NoSuchKey",
			expected: false,
		},
		{
			name:     "empty string",
			str:      "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substring",
			str:      "test string",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Integration tests (require AWS credentials and bucket)
func TestS3Client_Integration(t *testing.T) {
	if !shouldRunIntegrationTests(t) {
		t.Skip("Skipping integration tests - run with config file containing S3 bucket")
	}

	cfg := getTestConfig(t)
	client, err := NewS3Client(cfg)
	if err != nil {
		t.Fatalf("NewS3Client() error = %v", err)
	}

	// Test upload with test prefix to avoid conflicts
	testKeyWithPrefix := "integration-test/" + testKey
	t.Run("UploadFile", func(t *testing.T) {
		err := client.UploadFile(testData, testKeyWithPrefix)
		if err != nil {
			t.Errorf("UploadFile() error = %v", err)
		}
	})

	// Test file exists
	t.Run("FileExists", func(t *testing.T) {
		exists, err := client.FileExists(testKeyWithPrefix)
		if err != nil {
			t.Errorf("FileExists() error = %v", err)
		}
		if !exists {
			t.Error("FileExists() = false, want true")
		}
	})

	// Test download
	t.Run("DownloadFile", func(t *testing.T) {
		data, err := client.DownloadFile(testKeyWithPrefix)
		if err != nil {
			t.Errorf("DownloadFile() error = %v", err)
		}
		if !bytes.Equal(data, testData) {
			t.Errorf("DownloadFile() = %q, want %q", string(data), string(testData))
		}
	})

	// Test download non-existent file
	t.Run("DownloadFile_NotFound", func(t *testing.T) {
		data, err := client.DownloadFile("integration-test/non-existent-key")
		if err != nil {
			t.Errorf("DownloadFile() should not error for missing file, got: %v", err)
		}
		if data != nil {
			t.Errorf("DownloadFile() should return nil for missing file, got: %v", data)
		}
	})

	// Test file exists for non-existent file
	t.Run("FileExists_NotFound", func(t *testing.T) {
		exists, err := client.FileExists("integration-test/non-existent-key")
		if err != nil {
			t.Errorf("FileExists() error = %v", err)
		}
		if exists {
			t.Error("FileExists() = true, want false for non-existent file")
		}
	})

	// Cleanup
	t.Cleanup(func() {
		// Clean up test files
		if err := client.DeleteFile(testKeyWithPrefix); err != nil {
			t.Logf("Warning: Failed to cleanup test file %s: %v", testKeyWithPrefix, err)
		} else {
			t.Logf("Cleaned up test file: %s", testKeyWithPrefix)
		}
	})
}

// Benchmark tests
func BenchmarkUploadFile(b *testing.B) {
	// Check if we're running in short mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Try to load config to see if we have S3 credentials
	configPath := "../../test_config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../../secrets.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		b.Skipf("Skipping benchmark - cannot load config: %v", err)
	}

	if cfg.AWS.S3Bucket == "" {
		b.Skip("Skipping benchmark - no S3 bucket configured")
	}

	client, err := NewS3Client(&cfg.AWS)
	if err != nil {
		b.Fatalf("NewS3Client() error = %v", err)
	}

	data := make([]byte, 1024) // 1KB test data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("integration-test/benchmark-test-%d", i)
		err := client.UploadFile(data, key)
		if err != nil {
			b.Errorf("UploadFile() error = %v", err)
		}
	}
}