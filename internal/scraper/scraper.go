package scraper

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Scraper handles web scraping operations
type Scraper struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new scraper instance
func New() *Scraper {
	// Detect if we're running in AWS Lambda environment
	isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("CONFIG_PATH") == "/opt/secrets.yaml"

	var opts []chromedp.ExecAllocatorOption

	if isLambda {
		// Lambda/Container environment - use restrictive flags
		opts = []chromedp.ExecAllocatorOption{
			chromedp.NoDefaultBrowserCheck,
			chromedp.NoFirstRun,
			chromedp.Headless,
			// Lambda-specific flags for running Chrome in containers
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-setuid-sandbox", true),
			chromedp.Flag("no-zygote", true),
			chromedp.Flag("single-process", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-plugins", true),
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("disable-default-apps", true),
			chromedp.Flag("disable-sync", true),
			chromedp.Flag("disable-translate", true),
			chromedp.Flag("hide-scrollbars", true),
			chromedp.Flag("mute-audio", true),
			chromedp.Flag("no-first-run", true),
			chromedp.Flag("safebrowsing-disable-auto-update", true),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-features", "TranslateUI,VizDisplayCompositor"),
			chromedp.Flag("run-all-compositor-stages-before-draw", true),
			chromedp.Flag("memory-pressure-off", true),
			// Suppress cookie and other debug messages
			chromedp.Flag("log-level", "3"), // Only show fatal errors
		}
	} else {
		// Local development environment - use default options with minimal additions
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			// Essential flags that work well locally
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("hide-scrollbars", true),
			chromedp.Flag("mute-audio", true),
			// Suppress cookie and other debug messages
			chromedp.Flag("log-level", "3"), // Only show fatal errors
		)
	}

	// Create allocator context with custom options
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Create browser context with disabled logging and longer timeout
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...interface{}) {}))

	return &Scraper{
		ctx: ctx,
		cancel: func() {
			cancel()
			allocCancel()
		},
	}
}

// Close cleans up the scraper resources
func (s *Scraper) Close() {
	s.cancel()
}

// ScrapePageResult contains the results of scraping a page
type ScrapePageResult struct {
	HTML       string
	Screenshot []byte
	URL        string
	Timestamp  time.Time
}

// ScrapePage scrapes a webpage and returns both HTML content and a screenshot
func (s *Scraper) ScrapePage(url string) (*ScrapePageResult, error) {
	var html string
	var screenshot []byte

	// Create a fresh context for each scrape to avoid issues with context reuse
	ctx, cancel := chromedp.NewContext(s.ctx)
	defer cancel()

	// Add timeout to prevent hanging
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 45*time.Second)
	defer timeoutCancel()

	// Navigate to the page and capture both HTML and screenshot
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(url),
		chromedp.EmulateViewport(1200, 800, chromedp.EmulateScale(2)),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait for dynamic content to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Extract HTML that corresponds to the screenshot area
			var screenshotHTML string

			// First get the screenshot Y position
			var scheduleY float64 = 200 // Default fallback position
			var result map[string]interface{}
			err := chromedp.Evaluate(`
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
								viewportTop: rect.top,
								documentTop: rect.top + scrollY
							};
						}
					}
					return {found: false, fallback: 200};
				})()
			`, &result).Do(ctx)

			if err == nil && result["found"] == true {
				if documentTop, ok := result["documentTop"].(float64); ok {
					scheduleY = documentTop
				}
			}

			// Now extract HTML elements that fall within the screenshot bounds
			scriptWithY := fmt.Sprintf(`
				(function() {
					const screenshotTop = %f;
					const screenshotBottom = screenshotTop + 470; // Screenshot height
					const screenshotLeft = 150; // Screenshot X position  
					const screenshotRight = screenshotLeft + 340; // Screenshot width
					
					// First, find the H5 element with "Upcoming Schedule" text
					let upcomingScheduleElement = null;
					const allElements = document.querySelectorAll('*');
					
					for (let element of allElements) {
						if (element.tagName === 'H5' && element.textContent.trim().includes('Upcoming Schedule')) {
							upcomingScheduleElement = element;
							break;
						}
					}
					
					if (!upcomingScheduleElement) {
						// Fallback: try any element with "Upcoming Schedule"
						for (let element of allElements) {
							const text = element.textContent.trim();
							if ((text.startsWith('Upcoming Schedule') || text === 'Upcoming Schedule') && 
								element.children.length === 0) {
								upcomingScheduleElement = element;
								break;
							}
						}
					}
					
					if (!upcomingScheduleElement) {
						return '<!-- No Upcoming Schedule heading found -->';
					}
					
					// Get the position of the Upcoming Schedule element
					const scheduleRect = upcomingScheduleElement.getBoundingClientRect();
					const scheduleElementTop = scheduleRect.top + (window.pageYOffset || document.documentElement.scrollTop);
					
					// Get all elements that come after the Upcoming Schedule element
					const elementsAfterSchedule = [];
					
					for (let element of allElements) {
						const rect = element.getBoundingClientRect();
						const elementTop = rect.top + (window.pageYOffset || document.documentElement.scrollTop);
						const elementBottom = elementTop + rect.height;
						const elementLeft = rect.left;
						const elementRight = elementLeft + rect.width;
						
						// Only include elements that:
						// 1. Come after the "Upcoming Schedule" heading
						// 2. Are within the screenshot bounds
						// 3. Have meaningful content
						if (elementTop > scheduleElementTop) {
							const centerY = (elementTop + elementBottom) / 2;
							const centerX = (elementLeft + elementRight) / 2;
							
							if (centerY >= screenshotTop && centerY <= screenshotBottom &&
								centerX >= screenshotLeft && centerX <= screenshotRight) {
								
								const text = element.textContent.trim();
								const hasText = text.length > 0 && text.length < 500;
								const isRelevant = text.includes('game') || 
												  text.includes('practice') ||
												  text.includes('vs') ||
												  text.includes('pm') ||
												  text.includes('am') ||
												  text.includes('tbd') ||
												  text.includes('field') ||
												  /\d+:\d+/.test(text) ||
												  /\d+\/\d+/.test(text);
								
								if (hasText && (isRelevant || element.children.length === 0)) {
									elementsAfterSchedule.push(element);
								}
							}
						}
					}
					
					// Remove nested elements (keep only the most specific ones)
					const filteredElements = [];
					for (let element of elementsAfterSchedule) {
						let isNested = false;
						for (let other of elementsAfterSchedule) {
							if (other !== element && other.contains(element)) {
								isNested = true;
								break;
							}
						}
						if (!isNested) {
							filteredElements.push(element);
						}
					}
					
					// Create container with elements that come after "Upcoming Schedule"
					if (filteredElements.length > 0) {
						let container = document.createElement('div');
						container.className = 'schedule-content-after-heading';
						
						for (let element of filteredElements) {
							container.appendChild(element.cloneNode(true));
						}
						
						return container.innerHTML;
					}
					
					return '';
				})()
			`, scheduleY)

			err = chromedp.Evaluate(scriptWithY, &screenshotHTML).Do(ctx)

			if err == nil && screenshotHTML != "" {
				html = screenshotHTML
			} else {
				// Final fallback to full HTML
				return chromedp.OuterHTML("html", &html).Do(ctx)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Find the "Upcoming Schedule" text element and get its position
			var scheduleY float64 = 200 // Default fallback position

			// Use JavaScript to find the element with "Upcoming Schedule" text
			var result map[string]interface{}
			err := chromedp.Evaluate(`
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
						// Look for text that starts with "Upcoming Schedule" or is exactly "Upcoming Schedule"
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
								text: text
							};
						}
					}
					return {found: false, fallback: 200};
				})()
			`, &result).Do(ctx)

			if err == nil && result["found"] == true {
				// Use document-relative position (viewport + scroll)
				if documentTop, ok := result["documentTop"].(float64); ok {
					scheduleY = documentTop
				}
			}

			// Capture screenshot with dynamic clipping based on "Upcoming Schedule" position
			buf, err := page.CaptureScreenshot().
				WithClip(&page.Viewport{
					X:      150,
					Y:      scheduleY,
					Width:  340,
					Height: 470,
					Scale:  1,
				}).
				Do(ctx)
			if err != nil {
				return err
			}
			screenshot = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape page %s: %w", url, err)
	}

	return &ScrapePageResult{
		HTML:       html,
		Screenshot: screenshot,
		URL:        url,
		Timestamp:  time.Now(),
	}, nil
}

// SanitizeText cleans up problematic characters from text
func SanitizeText(text string) string {
	if text == "" {
		return text
	}

	// Remove invisible characters like U+200B
	text = strings.ReplaceAll(text, "\u200B", "")
	text = strings.ReplaceAll(text, "\u200C", "")
	text = strings.ReplaceAll(text, "\u200D", "")
	text = strings.ReplaceAll(text, "\uFEFF", "")

	// Replace en-dash with regular dash
	text = strings.ReplaceAll(text, "–", "-")

	return strings.TrimSpace(text)
}
