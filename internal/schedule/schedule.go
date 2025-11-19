package schedule

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"bandits-notification/internal/scraper"
	"bandits-notification/internal/storage"

	"github.com/PuerkitoBio/goquery"
)

// ScheduleEntry represents a single schedule entry
type ScheduleEntry struct {
	DayOfWeek  string     `json:"dayOfWeek"`
	DayOfMonth string     `json:"dayOfMonth"`
	Location   string     `json:"location"`
	TimeBlock  string     `json:"timeBlock"`
	Purpose    string     `json:"purpose,omitempty"`
	ParsedTime *time.Time `json:"parsedTime,omitempty"`
}

// Schedule represents a collection of schedule entries
type Schedule map[string]*ScheduleEntry

// ScheduleDiff represents the differences between two schedules
type ScheduleDiff struct {
	Added     Schedule `json:"added"`
	Deleted   Schedule `json:"deleted"`
	Modified  Schedule `json:"modified"`
	Unchanged Schedule `json:"unchanged"`
}

// ParseSchedule parses HTML content from the scraper and returns a Schedule object
func ParseSchedule(htmlContent string) (Schedule, error) {
	if htmlContent == "" {
		return make(Schedule), nil
	}

	// Use goquery to parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		// If HTML parsing fails, fall back to text parsing
		return parseScheduleFromText(htmlContent)
	}

	schedule := make(Schedule)

	// Find all elements and process them in order
	elements := make([]scheduleElement, 0)

	// Extract date headers and activity details
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Remove zero-width spaces that may appear in the HTML
		text = strings.ReplaceAll(text, "\u200b", "")
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		// Check if this is a date header (matches pattern: DAYNAME, M/D)
		dayPattern := regexp.MustCompile(`^(SUNDAY|MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY),\s*(\d+/\d+)$`)
		if dayPattern.MatchString(text) {
			elements = append(elements, scheduleElement{
				elementType: "date",
				text:        text,
			})
		} else {
			// Check for simplified date format (e.g., "11/3: BTC, 5:30–7:30" or "11/19 : BTC, 6:00-7:30")
			// This must be checked BEFORE the activity pattern since it's more specific
			// Note: Some entries have space before the colon (e.g., "11/19 :")
			simpleDatePattern := regexp.MustCompile(`^(\d+/\d+)\s*:\s*(.+)$`)
			if match := simpleDatePattern.FindStringSubmatch(text); len(match) == 3 {
				elements = append(elements, scheduleElement{
					elementType: "simple-date",
					text:        text,
				})
			} else if strings.Contains(text, ",") && (strings.Contains(text, ":") ||
				strings.Contains(strings.ToLower(text), "practice") ||
				strings.Contains(strings.ToLower(text), "game")) {
				// This looks like activity details
				elements = append(elements, scheduleElement{
					elementType: "activity",
					text:        text,
				})
			}
		}
	})

	// Process elements to build schedule entries
	var currentDate string
	for _, element := range elements {
		if element.elementType == "date" {
			currentDate = element.text
		} else if element.elementType == "activity" && currentDate != "" {
			// Parse the date
			dayPattern := regexp.MustCompile(`(SUNDAY|MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY),\s*(\d+/\d+)`)
			dateMatch := dayPattern.FindStringSubmatch(currentDate)
			if len(dateMatch) >= 3 {
				dayOfWeek := strings.ToUpper(dateMatch[1])
				dayOfMonth := dateMatch[2]

				// Parse activity details
				purpose, location, timeBlock := parseEventText(element.text)

				// Create the entry
				entry := &ScheduleEntry{
					DayOfWeek:  dayOfWeek,
					DayOfMonth: dayOfMonth,
					Location:   location,
					TimeBlock:  timeBlock,
					Purpose:    purpose,
				}

				// Try to parse the time
				if timeBlock != "" {
					parsedTime := parseTime(dayOfMonth, timeBlock)
					if parsedTime != nil {
						entry.ParsedTime = parsedTime
					}
				}

				// Use the date as the key
				key := fmt.Sprintf("%s, %s", dayOfWeek, dayOfMonth)
				schedule[key] = entry
			}
		} else if element.elementType == "simple-date" {
			// Handle simplified date format (e.g., "11/3: BTC, 5:30–7:30" or "11/19 : BTC, 6:00-7:30")
			simpleDatePattern := regexp.MustCompile(`^(\d+/\d+)\s*:\s*(.+)$`)
			match := simpleDatePattern.FindStringSubmatch(element.text)
			if len(match) == 3 {
				dayOfMonth := match[1]
				activityText := match[2]

				// Parse activity details
				purpose, location, timeBlock := parseEventText(activityText)

				// Create the entry (without day of week since it's not provided)
				entry := &ScheduleEntry{
					DayOfWeek:  "", // Not provided in this format
					DayOfMonth: dayOfMonth,
					Location:   location,
					TimeBlock:  timeBlock,
					Purpose:    purpose,
				}

				// Try to parse the time
				if timeBlock != "" {
					parsedTime := parseTime(dayOfMonth, timeBlock)
					if parsedTime != nil {
						entry.ParsedTime = parsedTime
					}
				}

				// Use the date as the key (without day of week)
				key := dayOfMonth
				schedule[key] = entry
			}
		}
	}

	return schedule, nil
}

// scheduleElement represents an element found during parsing
type scheduleElement struct {
	elementType string // "date" or "activity"
	text        string
}

// parseEventText extracts purpose, location, and time block from event text
// Handles both HTML activity text and plain text schedule entries
func parseEventText(text string) (purpose, location, timeBlock string) {
	text = strings.TrimSpace(scraper.SanitizeText(text))

	// Look for time patterns like "3:30–6:00", "10:30am", or "10:30"
	timeRegex := regexp.MustCompile(`\d+:\d+([-–—]\d+:\d+)?(am|pm)?`)
	timeMatch := timeRegex.FindString(text)

	if timeMatch != "" {
		timeBlock = timeMatch
		// Remove am/pm for consistency
		timeBlock = strings.TrimSuffix(strings.TrimSuffix(timeBlock, "am"), "pm")
		// Replace various dash characters with standard dash
		timeBlock = strings.ReplaceAll(timeBlock, "–", "-")
		timeBlock = strings.ReplaceAll(timeBlock, "—", "-")

		// Everything before the time block contains purpose and location
		beforeTime := strings.Split(text, timeMatch)[0]
		beforeTime = strings.TrimSpace(strings.TrimSuffix(beforeTime, ","))
		parts := strings.Split(beforeTime, ",")
		// Filter out empty parts
		var filteredParts []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				filteredParts = append(filteredParts, trimmed)
			}
		}
		parts = filteredParts
		if len(parts) >= 2 {
			// Last part is location, everything before is purpose
			location = strings.TrimSpace(parts[len(parts)-1])
			purposeParts := parts[:len(parts)-1]
			purpose = strings.TrimSpace(strings.Join(purposeParts, ", "))
		} else if len(parts) == 1 {
			// Only one part - assume it's the location
			location = strings.TrimSpace(parts[0])
		}
	} else {
		// No time block found, parse purpose and location from comma-separated parts
		parts := strings.Split(text, ",")
		// Filter out empty parts
		var filteredParts []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				filteredParts = append(filteredParts, trimmed)
			}
		}
		parts = filteredParts

		if len(parts) >= 2 {
			// Last part is location, everything before is purpose
			location = strings.TrimSpace(parts[len(parts)-1])
			purposeParts := parts[:len(parts)-1]
			purpose = strings.TrimSpace(strings.Join(purposeParts, ", "))
		} else {
			// Single part - assume it's the location
			location = strings.TrimSpace(text)
		}
	}

	return purpose, location, timeBlock
}

// parseScheduleFromText is a fallback parser for plain text
func parseScheduleFromText(text string) (Schedule, error) {
	text = scraper.SanitizeText(text)

	schedule := make(Schedule)

	// First, try to parse traditional format with day names
	dayPattern := regexp.MustCompile(`(?i)(SUNDAY|MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY),\s*(\d+/\d+)`)
	dayMatches := dayPattern.FindAllStringSubmatch(text, -1)
	dayIndices := dayPattern.FindAllStringSubmatchIndex(text, -1)

	for i, match := range dayMatches {
		if len(match) < 3 {
			continue
		}

		dayOfWeek := strings.ToUpper(match[1])
		dayOfMonth := match[2]

		// Get the content between this day and the next day (or end of string)
		var content string
		startIdx := dayIndices[i][1] // End of current match
		if i+1 < len(dayIndices) {
			endIdx := dayIndices[i+1][0] // Start of next match
			content = text[startIdx:endIdx]
		} else {
			content = text[startIdx:]
		}

		// Extract purpose, location, and time from content
		purpose, location, timeBlock := parseEventText(content)

		// If no meaningful content was found, skip this entry
		if location == "" && timeBlock == "" {
			continue
		}

		// Create the entry
		entry := &ScheduleEntry{
			DayOfWeek:  dayOfWeek,
			DayOfMonth: dayOfMonth,
			Location:   location,
			TimeBlock:  timeBlock,
			Purpose:    purpose,
		}

		// Try to parse the time
		if timeBlock != "" {
			parsedTime := parseTime(dayOfMonth, timeBlock)
			if parsedTime != nil {
				entry.ParsedTime = parsedTime
			}
		}

		// Use the full match as the key
		key := fmt.Sprintf("%s, %s", dayOfWeek, dayOfMonth)
		schedule[key] = entry
	}

	// Also parse simplified date format (e.g., "11/3: BTC, 5:30–7:30" or "11/19 : BTC, 6:00-7:30")
	// Clean zero-width spaces first
	text = strings.ReplaceAll(text, "\u200b", "")
	simpleDatePattern := regexp.MustCompile(`(?m)^(\d+/\d+)\s*:\s*(.+)$`)
	simpleDateMatches := simpleDatePattern.FindAllStringSubmatch(text, -1)

	for _, match := range simpleDateMatches {
		if len(match) < 3 {
			continue
		}

		dayOfMonth := match[1]
		activityText := match[2]

		// Parse activity details
		purpose, location, timeBlock := parseEventText(activityText)

		// If no meaningful content was found, skip this entry
		if location == "" && timeBlock == "" {
			continue
		}

		// Create the entry
		entry := &ScheduleEntry{
			DayOfWeek:  "", // Not provided in this format
			DayOfMonth: dayOfMonth,
			Location:   location,
			TimeBlock:  timeBlock,
			Purpose:    purpose,
		}

		// Try to parse the time
		if timeBlock != "" {
			parsedTime := parseTime(dayOfMonth, timeBlock)
			if parsedTime != nil {
				entry.ParsedTime = parsedTime
			}
		}

		// Use the date as the key (without day of week)
		key := dayOfMonth
		schedule[key] = entry
	}

	return schedule, nil
}

// parseTime attempts to parse a time string into a time.Time
func parseTime(dayOfMonth, timeBlock string) *time.Time {
	if timeBlock == "" || dayOfMonth == "" {
		return nil
	}

	// This is a simplified time parsing - you may want to use a more sophisticated
	// time parsing library like the Go equivalent of chrono-node
	// For now, we'll just store the time block as-is

	// You could implement more sophisticated parsing here using time.Parse
	// with various formats, or integrate a natural language date parsing library

	return nil
}

// CompareSchedules compares two schedules and returns the differences
func CompareSchedules(old, new Schedule) *ScheduleDiff {
	diff := &ScheduleDiff{
		Added:     make(Schedule),
		Deleted:   make(Schedule),
		Modified:  make(Schedule),
		Unchanged: make(Schedule),
	}

	if old == nil {
		// If old schedule is nil, everything is added
		for key, entry := range new {
			diff.Added[key] = entry
		}
		return diff
	}

	// Find deleted entries
	for key, entry := range old {
		if _, exists := new[key]; !exists {
			diff.Deleted[key] = entry
		}
	}

	// Find added and modified entries
	for key, newEntry := range new {
		oldEntry, exists := old[key]
		if !exists {
			diff.Added[key] = newEntry
		} else {
			// Check if modified
			if oldEntry.Location != newEntry.Location || oldEntry.TimeBlock != newEntry.TimeBlock || oldEntry.Purpose != newEntry.Purpose {
				diff.Modified[key] = newEntry
			} else {
				diff.Unchanged[key] = newEntry
			}
		}
	}

	return diff
}

// HasChanges returns true if the schedule diff contains any changes
func (d *ScheduleDiff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Deleted) > 0 || len(d.Modified) > 0
}

// SerializeSchedule converts a schedule to JSON bytes for storage
func SerializeSchedule(schedule Schedule) ([]byte, error) {
	return json.Marshal(schedule)
}

// DeserializeSchedule converts JSON bytes back to a Schedule
func DeserializeSchedule(data []byte) (Schedule, error) {
	if len(data) == 0 {
		return make(Schedule), nil
	}

	var schedule Schedule
	if err := json.Unmarshal(data, &schedule); err != nil {
		return nil, fmt.Errorf("failed to deserialize schedule: %w", err)
	}

	return schedule, nil
}

// SaveSchedule saves a schedule to storage with urlIdentifier-based key
func SaveSchedule(s3Client *storage.S3Client, schedule Schedule, urlIdentifier, filename string) error {
	data, err := SerializeSchedule(schedule)
	if err != nil {
		return fmt.Errorf("failed to serialize schedule: %w", err)
	}

	key := urlIdentifier + "/" + filename
	return s3Client.UploadFile(data, key)
}

// LoadSchedule loads a schedule from storage with urlIdentifier-based key
func LoadSchedule(s3Client *storage.S3Client, urlIdentifier, filename string) (Schedule, error) {
	key := urlIdentifier + "/" + filename
	data, err := s3Client.DownloadFile(key)
	if err != nil {
		return nil, fmt.Errorf("failed to download schedule: %w", err)
	}

	return DeserializeSchedule(data)
}

// GetTimestampedFilename generates a timestamped filename
func GetTimestampedFilename(base, extension string) string {
	now := time.Now()
	timestamp := now.Unix()
	return fmt.Sprintf("%s-%d-%d-%d-%d.%s",
		base,
		now.Year(),
		int(now.Month()),
		now.Day(),
		timestamp,
		extension)
}

// GetURLIdentifier extracts the last path segment from a URL to use as S3 key prefix
// For example: "https://www.brooklinebaseball.net/bandits12u" -> "bandits12u"
// If isTest is true, appends "-test" suffix
func GetURLIdentifier(urlStr string, isTest bool) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Fallback to simple string manipulation if URL parsing fails
		parts := strings.Split(strings.TrimSuffix(urlStr, "/"), "/")
		if len(parts) > 0 {
			identifier := parts[len(parts)-1]
			if identifier == "" && len(parts) > 1 {
				identifier = parts[len(parts)-2]
			}
			if isTest {
				return identifier + "-test"
			}
			return identifier
		}
		return "unknown"
	}

	// Extract the last non-empty path segment
	pathSegments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	identifier := "unknown"
	for i := len(pathSegments) - 1; i >= 0; i-- {
		if pathSegments[i] != "" {
			identifier = pathSegments[i]
			break
		}
	}

	if isTest {
		return identifier + "-test"
	}
	return identifier
}
