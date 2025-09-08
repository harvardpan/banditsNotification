package scraper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes invisible characters",
			input:    "Hello\u200BWorld\u200C\u200D\uFEFF",
			expected: "HelloWorld",
		},
		{
			name:     "replaces en-dash with regular dash",
			input:    "Time: 3:00–5:00",
			expected: "Time: 3:00-5:00",
		},
		{
			name:     "trims whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles normal text",
			input:    "Normal text with spaces",
			expected: "Normal text with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeText(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Integration test for Bandits 12U site
func TestScrapePage_Integration_12U(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	scraper := New()
	defer scraper.Close()

	// Test with the Brookline Bandits 12U schedule page
	result, err := scraper.ScrapePage("https://www.brooklinebaseball.net/bandits12u")
	if err != nil {
		t.Fatalf("ScrapePage() error = %v", err)
	}

	// Basic validation
	if result == nil {
		t.Fatal("ScrapePage() returned nil result")
	}

	if result.URL != "https://www.brooklinebaseball.net/bandits12u" {
		t.Errorf("ScrapePage() URL = %q, want %q", result.URL, "https://www.brooklinebaseball.net/bandits12u")
	}

	// Create test output directory
	outputDir := "test-output"
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Errorf("Failed to create output directory: %v", err)
	}

	// Save screenshot to PNG file for inspection
	screenshotPath := filepath.Join(outputDir, "test_12u_screenshot.png")
	err = os.WriteFile(screenshotPath, result.Screenshot, 0644)
	if err != nil {
		t.Errorf("Failed to save 12U screenshot: %v", err)
	} else {
		t.Logf("12U Screenshot saved to %s", screenshotPath)
	}

	// Save scraped HTML to file for inspection
	htmlPath := filepath.Join(outputDir, "test_12u_scraped_content.html")
	htmlContent := createTestHTMLFile(result.HTML, result.URL, "12U")

	err = os.WriteFile(htmlPath, []byte(htmlContent), 0644)
	if err != nil {
		t.Errorf("Failed to save 12U HTML content: %v", err)
	} else {
		t.Logf("12U Scraped HTML content saved to %s", htmlPath)
	}

	// Verify content
	validateScheduleContent(t, result, "12U")
}

// Integration test for Bandits 14U site
func TestScrapePage_Integration_14U(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	scraper := New()
	defer scraper.Close()

	// Test with the Brookline Bandits 14U schedule page
	result, err := scraper.ScrapePage("https://www.brooklinebaseball.net/bandits14u")
	if err != nil {
		t.Fatalf("ScrapePage() error = %v", err)
	}

	// Basic validation
	if result == nil {
		t.Fatal("ScrapePage() returned nil result")
	}

	if result.URL != "https://www.brooklinebaseball.net/bandits14u" {
		t.Errorf("ScrapePage() URL = %q, want %q", result.URL, "https://www.brooklinebaseball.net/bandits14u")
	}

	// Create test output directory
	outputDir := "test-output"
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Errorf("Failed to create output directory: %v", err)
	}

	// Save screenshot to PNG file for inspection
	screenshotPath := filepath.Join(outputDir, "test_14u_screenshot.png")
	err = os.WriteFile(screenshotPath, result.Screenshot, 0644)
	if err != nil {
		t.Errorf("Failed to save 14U screenshot: %v", err)
	} else {
		t.Logf("14U Screenshot saved to %s", screenshotPath)
	}

	// Save scraped HTML to file for inspection
	htmlPath := filepath.Join(outputDir, "test_14u_scraped_content.html")
	htmlContent := createTestHTMLFile(result.HTML, result.URL, "14U")

	err = os.WriteFile(htmlPath, []byte(htmlContent), 0644)
	if err != nil {
		t.Errorf("Failed to save 14U HTML content: %v", err)
	} else {
		t.Logf("14U Scraped HTML content saved to %s", htmlPath)
	}

	// Verify content
	validateScheduleContent(t, result, "14U")
}

// Helper function to create test HTML file with styling
func createTestHTMLFile(htmlContent, url, team string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scraped Schedule Content - Bandits %s</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 20px; 
            background-color: #f5f5f5; 
        }
        .header { 
            background: #333; 
            color: white; 
            padding: 10px; 
            border-radius: 5px; 
            margin-bottom: 20px; 
        }
        .content { 
            background: white; 
            padding: 20px; 
            border-radius: 5px; 
            box-shadow: 0 2px 5px rgba(0,0,0,0.1); 
        }
        .stats {
            background: #e8f4fd;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 20px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Scraped Schedule Content - Bandits %s</h1>
        <p>This HTML represents the content extracted from the 340x470 pixel screenshot area</p>
    </div>
    
    <div class="stats">
        <strong>Content Statistics:</strong><br>
        • HTML Size: %d bytes<br>
        • Source URL: %s<br>
        • Team: Bandits %s<br>
        • Screenshot Dimensions: 340x470 pixels<br>
        • Extraction Method: Elements after "Upcoming Schedule" heading within screenshot bounds
    </div>
    
    <div class="content">
        <h2>Extracted Content:</h2>
        %s
    </div>
</body>
</html>`, team, team, len(htmlContent), url, team, htmlContent)
}

// Helper function to validate schedule content
func validateScheduleContent(t *testing.T, result *ScrapePageResult, team string) {
	// Verify HTML contains expected content for Bandits schedule page
	if !strings.Contains(strings.ToLower(result.HTML), "bandits") {
		t.Errorf("%s: ScrapePage() HTML doesn't contain expected Bandits content", team)
	}

	// Verify that we got focused HTML (should be much smaller than full page)
	if len(result.HTML) > 50000 { // Full page is typically much larger
		t.Logf("%s: Warning: HTML size is %d bytes, might not be properly focused on schedule section", team, len(result.HTML))
	} else {
		t.Logf("%s: HTML focused on schedule section: %d bytes", team, len(result.HTML))
	}

	// Verify the HTML contains schedule-related content (but not the heading itself)
	htmlLower := strings.ToLower(result.HTML)
	hasScheduleContent := strings.Contains(htmlLower, "game") ||
		strings.Contains(htmlLower, "practice") ||
		strings.Contains(htmlLower, "vs") ||
		strings.Contains(htmlLower, "pm") ||
		strings.Contains(htmlLower, "field") ||
		strings.Contains(htmlLower, "tbd")

	if !hasScheduleContent {
		t.Errorf("%s: ScrapePage() HTML doesn't contain expected schedule content (games, practices, etc.)", team)
	}

	// Verify screenshot is not empty
	if len(result.Screenshot) == 0 {
		t.Errorf("%s: Screenshot is empty", team)
	} else {
		t.Logf("%s: Screenshot size: %d bytes", team, len(result.Screenshot))
	}
}

// Test to debug getBoundingClientRect positioning
func TestDebugSchedulePosition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test")
	}

	scraper := New()
	defer scraper.Close()

	// First navigate to the page
	err := chromedp.Run(scraper.ctx,
		chromedp.Navigate("https://www.brooklinebaseball.net/bandits12u"),
		chromedp.EmulateViewport(1200, 800, chromedp.EmulateScale(2)),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to navigate to page: %v", err)
	}

	// Extract debugging info about the "Upcoming Schedule" element
	var debugInfo map[string]interface{}
	err = chromedp.Run(scraper.ctx, chromedp.Evaluate(`
		(function() {
			const walker = document.createTreeWalker(
				document.body,
				NodeFilter.SHOW_TEXT,
				null,
				false
			);
			
			let node;
			while (node = walker.nextNode()) {
				const text = node.textContent.trim();
				if (text.startsWith('Upcoming Schedule') || text === 'Upcoming Schedule') {
					const rect = node.parentElement.getBoundingClientRect();
					const scrollY = window.pageYOffset || document.documentElement.scrollTop;
					return {
						found: true,
						elementTag: node.parentElement.tagName,
						elementClass: node.parentElement.className,
						viewportTop: rect.top,
						documentTop: rect.top + scrollY,
						scrollY: scrollY,
						windowHeight: window.innerHeight,
						viewportHeight: document.documentElement.clientHeight,
						text: text.substring(0, 50)
					};
				}
			}
			return {found: false};
		})()
	`, &debugInfo))

	if err != nil {
		t.Fatalf("Failed to evaluate debug JS: %v", err)
	}

	if debugInfo["found"] == true {
		fmt.Printf("=== Upcoming Schedule Element Debug Info ===\n")
		fmt.Printf("Element Tag: %v\n", debugInfo["elementTag"])
		fmt.Printf("Element Class: %v\n", debugInfo["elementClass"])
		fmt.Printf("Text Content: %v\n", debugInfo["text"])
		fmt.Printf("Viewport Top: %v\n", debugInfo["viewportTop"])
		fmt.Printf("Document Top: %v\n", debugInfo["documentTop"])
		fmt.Printf("Scroll Y: %v\n", debugInfo["scrollY"])
		fmt.Printf("Window Height: %v\n", debugInfo["windowHeight"])
		fmt.Printf("Viewport Height: %v\n", debugInfo["viewportHeight"])
		fmt.Printf("==========================================\n")

		t.Logf("Found 'Upcoming Schedule' at viewport position %v, document position %v",
			debugInfo["viewportTop"], debugInfo["documentTop"])
	} else {
		t.Log("Could not find 'Upcoming Schedule' text on the page")
	}
}

// Mock test for scraping without network
func TestScrapePage_Mock(t *testing.T) {
	// This test demonstrates how you might mock the scraping functionality
	// In a real implementation, you'd inject dependencies to make this testable

	t.Skip("Mock implementation would require dependency injection refactor")

	// Expected approach:
	// 1. Create an interface for the Chrome driver
	// 2. Inject the driver into Scraper
	// 3. Create a mock driver for testing
	// 4. Test the logic without actually launching Chrome
}
