package storage

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"bandits-notification/internal/config"
)

// S3Client wraps AWS S3 operations
type S3Client struct {
	client *s3.S3
	bucket string
}

// NewS3Client creates a new S3 client
func NewS3Client(cfg *config.AWSConfig) (*S3Client, error) {
	// Create AWS config
	awsConfig := &aws.Config{
		Region: aws.String(cfg.Region),
	}
	
	// If explicit credentials are provided, use them. Otherwise, use default credential chain
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"", // token
		)
	}
	// If no explicit credentials, AWS SDK will use default credential chain:
	// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// 2. Shared credentials file (~/.aws/credentials)
	// 3. IAM roles (for EC2 instances)
	// 4. SSO credentials (when logged in via `aws sso login`)
	
	// Create AWS session with profile support
	var sess *session.Session
	var err error
	
	// Check if AWS_PROFILE environment variable is set
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		sess, err = session.NewSessionWithOptions(session.Options{
			Config:  *awsConfig,
			Profile: profile,
		})
	} else {
		sess, err = session.NewSession(awsConfig)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Client{
		client: s3.New(sess),
		bucket: cfg.S3Bucket,
	}, nil
}

// UploadFile uploads data to S3 with the specified key
func (s *S3Client) UploadFile(data []byte, key string) error {
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

// DownloadFile downloads data from S3 with the specified key
func (s *S3Client) DownloadFile(key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if the error is because the key doesn't exist
		if isNoSuchKeyError(err) {
			return nil, nil // Return nil data for missing files (similar to original logic)
		}
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

// FileExists checks if a file exists in S3
func (s *S3Client) FileExists(key string) (bool, error) {
	_, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNoSuchKeyError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}
	return true, nil
}

// DeleteFile deletes a file from S3
func (s *S3Client) DeleteFile(key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

// isNoSuchKeyError checks if an error is a "NoSuchKey" error
func isNoSuchKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Check for AWS NoSuchKey error
	return contains(err.Error(), "NoSuchKey") || contains(err.Error(), "NotFound")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}