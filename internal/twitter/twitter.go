package twitter

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"bandits-notification/internal/config"
)

// Client wraps the Twitter API client functionality
type Client struct {
	config     *config.TwitterConfig
	httpClient *http.Client
}

// New creates a new Twitter client
func New(cfg *config.TwitterConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// User represents a Twitter user for verification
type User struct {
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
	ID         string `json:"id_str"`
}

// UploadMedia uploads image data to Twitter and returns media ID
func (c *Client) UploadMedia(imageData []byte) (string, error) {
	// Twitter's v1.1 media upload API expects base64 encoded data in form parameters
	encodedData := base64.StdEncoding.EncodeToString(imageData)

	params := url.Values{
		"media_data": []string{encodedData},
	}

	// Make authenticated request to media upload endpoint
	req, err := http.NewRequest("POST", "https://upload.twitter.com/1.1/media/upload.json", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add OAuth1 authorization header
	err = c.addOAuth1Header(req, params)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		MediaIDString string `json:"media_id_string"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.MediaIDString, nil
}

// VerifyCredentials checks if the Twitter credentials are valid
func (c *Client) VerifyCredentials() (*User, error) {
	req, err := http.NewRequest("GET", "https://api.twitter.com/1.1/account/verify_credentials.json", nil)
	if err != nil {
		return nil, err
	}

	// Add OAuth1 authorization header
	err = c.addOAuth1Header(req, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("credential verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[TWITTER] Verified credentials for @%s\n", user.ScreenName)
	return &user, nil
}

// addOAuth1Header adds OAuth1 authorization header to the request
func (c *Client) addOAuth1Header(req *http.Request, params url.Values) error {
	// Generate OAuth parameters
	nonce := generateNonce()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	oauthParams := map[string]string{
		"oauth_consumer_key":     c.config.ConsumerKey,
		"oauth_token":            c.config.AccessToken,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_nonce":            nonce,
		"oauth_version":          "1.0",
	}

	// Create signature
	signature, err := c.generateOAuth1Signature(req.Method, req.URL.String(), oauthParams, params)
	if err != nil {
		return err
	}
	oauthParams["oauth_signature"] = signature

	// Build authorization header (only OAuth parameters, not POST data)
	// Sort OAuth parameters for consistent ordering
	var oauthKeys []string
	for key := range oauthParams {
		if strings.HasPrefix(key, "oauth_") {
			oauthKeys = append(oauthKeys, key)
		}
	}
	sort.Strings(oauthKeys)

	var authParts []string
	for _, key := range oauthKeys {
		value := oauthParams[key]
		authParts = append(authParts, fmt.Sprintf(`%s="%s"`, oauthPercentEncode(key), oauthPercentEncode(value)))
	}

	authHeader := "OAuth " + strings.Join(authParts, ", ")
	req.Header.Set("Authorization", authHeader)

	return nil
}

// oauthPercentEncode implements Twitter's specific percent encoding
// This matches the Node.js implementation exactly
func oauthPercentEncode(str string) string {
	return strings.NewReplacer(
		"!", "%21",
		"*", "%2A",
		"'", "%27",
		"(", "%28",
		")", "%29",
	).Replace(url.QueryEscape(str))
}

// parseURL extracts query parameters from URL
func parseURL(fullURL string) (string, map[string]string) {
	parts := strings.Split(fullURL, "?")
	baseURL := parts[0]
	params := make(map[string]string)

	if len(parts) > 1 {
		for _, param := range strings.Split(parts[1], "&") {
			if kv := strings.Split(param, "="); len(kv) == 2 {
				key, _ := url.QueryUnescape(kv[0])
				value, _ := url.QueryUnescape(kv[1])
				params[key] = value
			}
		}
	}
	return baseURL, params
}

// generateOAuth1Signature creates OAuth1 signature matching Node.js implementation
func (c *Client) generateOAuth1Signature(method, fullURL string, oauthParams map[string]string, postParams url.Values) (string, error) {
	// Extract base URL and query parameters
	baseURL, urlParams := parseURL(fullURL)

	// Combine OAuth, URL query, and POST parameters
	allParams := make(map[string]string)

	// Add OAuth parameters
	for k, v := range oauthParams {
		allParams[k] = v
	}

	// Add URL query parameters
	for k, v := range urlParams {
		allParams[k] = v
	}

	// Add POST parameters
	for k, v := range postParams {
		if len(v) > 0 {
			allParams[k] = v[0]
		}
	}

	// Sort parameters by key
	var keys []string
	for k := range allParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build parameter string with Twitter's percent encoding
	var paramParts []string
	for _, k := range keys {
		encodedKey := oauthPercentEncode(k)
		encodedValue := oauthPercentEncode(allParams[k])
		paramParts = append(paramParts, fmt.Sprintf("%s=%s", encodedKey, encodedValue))
	}
	paramString := strings.Join(paramParts, "&")

	// Create signature base string
	signatureBase := fmt.Sprintf("%s&%s&%s",
		oauthPercentEncode(strings.ToUpper(method)),
		oauthPercentEncode(baseURL),
		oauthPercentEncode(paramString))

	// Create signing key
	signingKey := fmt.Sprintf("%s&%s",
		oauthPercentEncode(c.config.ConsumerSecret),
		oauthPercentEncode(c.config.AccessTokenSecret))

	// Generate HMAC-SHA1 signature
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(signatureBase))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature, nil
}

// PostTweetWithMediaAndReturnID posts a tweet with media using v2 API and returns the tweet ID
func (c *Client) PostTweetWithMediaAndReturnID(text string, mediaIDs []string) (string, error) {
	// Build request body for v2 API
	requestBody := map[string]interface{}{
		"text": text,
	}

	if len(mediaIDs) > 0 {
		requestBody["media"] = map[string]interface{}{
			"media_ids": mediaIDs,
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/tweets", strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Add OAuth1 authorization header (no form params for v2 API)
	err = c.addOAuth1Header(req, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tweet failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse v2 API response to get tweet ID
	var result struct {
		Data struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	fmt.Printf("[TWITTER] Successfully posted tweet: %s (ID: %s)\n", text, result.Data.ID)
	return result.Data.ID, nil
}

// DeleteTweet deletes a tweet by ID
func (c *Client) DeleteTweet(tweetID string) error {
	endpoint := fmt.Sprintf("https://api.twitter.com/1.1/statuses/destroy/%s.json", tweetID)

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}

	// Add OAuth1 authorization header
	err = c.addOAuth1Header(req, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete tweet failed with status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[TWITTER] Successfully deleted tweet: %s\n", tweetID)
	return nil
}

// generateNonce creates a random nonce for OAuth
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
