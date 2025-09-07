package config

import (
	"fmt"
	"time"

	"github.com/getsops/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	AWS AWSConfig `yaml:"aws"`
	App AppConfig `yaml:"app"`
}

// TwitterConfig holds Twitter API credentials
type TwitterConfig struct {
	ConsumerKey       string `yaml:"consumer_key"`
	ConsumerSecret    string `yaml:"consumer_secret"`
	AccessToken       string `yaml:"access_token"`
	AccessTokenSecret string `yaml:"access_token_secret"`
	UserHandle        string `yaml:"user_handle"`
}

// AWSConfig holds AWS credentials and settings
type AWSConfig struct {
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Region          string `yaml:"region"`
	S3Bucket        string `yaml:"s3_bucket"`
}

// URLConfig holds URL-specific configuration including Twitter credentials
type URLConfig struct {
	URL     string        `yaml:"url"`
	Twitter TwitterConfig `yaml:"twitter"`
}

// AppConfig holds application-specific settings
type AppConfig struct {
	DisplayTimezone string      `yaml:"display_timezone"`
	URLs            []URLConfig `yaml:"urls"`
}

// LoadConfig loads configuration from a SOPS-encrypted YAML file
func LoadConfig(configPath string) (*Config, error) {
	// Decrypt the SOPS file
	cleartext, err := decrypt.File(configPath, "yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config file: %w", err)
	}

	// Parse the decrypted YAML
	var config Config
	if err := yaml.Unmarshal(cleartext, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Set defaults
	if config.AWS.Region == "" {
		config.AWS.Region = "us-east-1"
	}
	if config.App.DisplayTimezone == "" {
		config.App.DisplayTimezone = "America/New_York"
	}

	return &config, nil
}

// GetTimezone returns the configured timezone location
func (c *Config) GetTimezone() (*time.Location, error) {
	return time.LoadLocation(c.App.DisplayTimezone)
}