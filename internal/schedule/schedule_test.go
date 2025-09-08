package schedule

import (
	"encoding/json"
	"testing"
	"time"
)

// Test data for schedule parsing - simulating the new HTML format from scraper
const sampleScheduleHTML = `<span style="color:#0B1C2F;" class="wixui-rich-text__text">Our plan for the week ahead.</span>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">MONDAY, 12/5</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Team Practice, Field A, 3:00-5:00</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">WEDNESDAY, 12/7</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Game vs Tigers, Field B, 6:00</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">FRIDAY, 12/9</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Team Practice, Home Field, 4:00-6:00</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">SATURDAY, 12/10</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Tournament Game, Away Field, 10:00</span></p>`

// Legacy text format for fallback testing
const sampleScheduleText = `
MONDAY, 12/5 Field A, 3:00-5:00pm Practice
WEDNESDAY, 12/7 Field B 6:00pm Game vs Tigers
FRIDAY, 12/9 Home Field, 4:00-6:00pm Practice  
SATURDAY, 12/10 Away Field 10:00am Tournament Game`

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedCount int
		expectedKeys  []string
		shouldContain map[string]*ScheduleEntry
	}{
		{
			name:          "parses HTML schedule from scraper",
			content:       sampleScheduleHTML,
			expectedCount: 4,
			expectedKeys:  []string{"MONDAY, 12/5", "WEDNESDAY, 12/7", "FRIDAY, 12/9", "SATURDAY, 12/10"},
			shouldContain: map[string]*ScheduleEntry{
				"MONDAY, 12/5": {
					DayOfWeek:  "MONDAY",
					DayOfMonth: "12/5",
					Location:   "Team Practice, Field A",
					TimeBlock:  "3:00-5:00",
				},
				"WEDNESDAY, 12/7": {
					DayOfWeek:  "WEDNESDAY",
					DayOfMonth: "12/7",
					Location:   "Game vs Tigers, Field B",
					TimeBlock:  "6:00",
				},
			},
		},
		{
			name:          "handles empty content",
			content:       "",
			expectedCount: 0,
			expectedKeys:  []string{},
		},
		{
			name:          "handles single entry HTML",
			content:       `<span>MONDAY, 12/5</span><span>Team Practice, Field A, 3:00</span>`,
			expectedCount: 1,
			expectedKeys:  []string{"MONDAY, 12/5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSchedule(tt.content)
			if err != nil {
				t.Errorf("ParseSchedule() error = %v", err)
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("ParseSchedule() count = %d, want %d", len(result), tt.expectedCount)
			}

			// Check that expected keys exist
			for _, key := range tt.expectedKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("ParseSchedule() missing expected key: %s", key)
				}
			}

			// Check specific entries
			for key, expected := range tt.shouldContain {
				actual, exists := result[key]
				if !exists {
					t.Errorf("ParseSchedule() missing key: %s", key)
					continue
				}

				if actual.DayOfWeek != expected.DayOfWeek {
					t.Errorf("ParseSchedule() key %s DayOfWeek = %s, want %s", key, actual.DayOfWeek, expected.DayOfWeek)
				}
				if actual.DayOfMonth != expected.DayOfMonth {
					t.Errorf("ParseSchedule() key %s DayOfMonth = %s, want %s", key, actual.DayOfMonth, expected.DayOfMonth)
				}
				if actual.Location != expected.Location {
					t.Errorf("ParseSchedule() key %s Location = %s, want %s", key, actual.Location, expected.Location)
				}
				if actual.TimeBlock != expected.TimeBlock {
					t.Errorf("ParseSchedule() key %s TimeBlock = %s, want %s", key, actual.TimeBlock, expected.TimeBlock)
				}
			}
		})
	}
}

func TestCompareSchedules(t *testing.T) {
	// Create test schedules
	oldSchedule := Schedule{
		"MONDAY, 12/5": {
			DayOfWeek:  "MONDAY",
			DayOfMonth: "12/5",
			Location:   "Field A",
			TimeBlock:  "3:00pm",
		},
		"WEDNESDAY, 12/7": {
			DayOfWeek:  "WEDNESDAY",
			DayOfMonth: "12/7",
			Location:   "Field B",
			TimeBlock:  "6:00pm",
		},
	}

	newSchedule := Schedule{
		"MONDAY, 12/5": {
			DayOfWeek:  "MONDAY",
			DayOfMonth: "12/5",
			Location:   "Field C", // Changed location
			TimeBlock:  "3:00pm",
		},
		"FRIDAY, 12/9": {
			DayOfWeek:  "FRIDAY",
			DayOfMonth: "12/9",
			Location:   "Home Field",
			TimeBlock:  "4:00pm",
		},
	}

	tests := []struct {
		name     string
		old      Schedule
		new      Schedule
		expected *ScheduleDiff
	}{
		{
			name: "detects all types of changes",
			old:  oldSchedule,
			new:  newSchedule,
			expected: &ScheduleDiff{
				Added: Schedule{
					"FRIDAY, 12/9": newSchedule["FRIDAY, 12/9"],
				},
				Deleted: Schedule{
					"WEDNESDAY, 12/7": oldSchedule["WEDNESDAY, 12/7"],
				},
				Modified: Schedule{
					"MONDAY, 12/5": newSchedule["MONDAY, 12/5"],
				},
				Unchanged: Schedule{},
			},
		},
		{
			name: "handles nil old schedule",
			old:  nil,
			new:  newSchedule,
			expected: &ScheduleDiff{
				Added:     newSchedule,
				Deleted:   Schedule{},
				Modified:  Schedule{},
				Unchanged: Schedule{},
			},
		},
		{
			name: "handles identical schedules",
			old:  oldSchedule,
			new:  oldSchedule,
			expected: &ScheduleDiff{
				Added:     Schedule{},
				Deleted:   Schedule{},
				Modified:  Schedule{},
				Unchanged: oldSchedule,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSchedules(tt.old, tt.new)

			if len(result.Added) != len(tt.expected.Added) {
				t.Errorf("CompareSchedules() Added count = %d, want %d", len(result.Added), len(tt.expected.Added))
			}
			if len(result.Deleted) != len(tt.expected.Deleted) {
				t.Errorf("CompareSchedules() Deleted count = %d, want %d", len(result.Deleted), len(tt.expected.Deleted))
			}
			if len(result.Modified) != len(tt.expected.Modified) {
				t.Errorf("CompareSchedules() Modified count = %d, want %d", len(result.Modified), len(tt.expected.Modified))
			}
			if len(result.Unchanged) != len(tt.expected.Unchanged) {
				t.Errorf("CompareSchedules() Unchanged count = %d, want %d", len(result.Unchanged), len(tt.expected.Unchanged))
			}

			// Verify specific entries exist in the right categories
			for key := range tt.expected.Added {
				if _, exists := result.Added[key]; !exists {
					t.Errorf("CompareSchedules() missing expected added key: %s", key)
				}
			}
			for key := range tt.expected.Deleted {
				if _, exists := result.Deleted[key]; !exists {
					t.Errorf("CompareSchedules() missing expected deleted key: %s", key)
				}
			}
			for key := range tt.expected.Modified {
				if _, exists := result.Modified[key]; !exists {
					t.Errorf("CompareSchedules() missing expected modified key: %s", key)
				}
			}
		})
	}
}

func TestScheduleDiff_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     *ScheduleDiff
		expected bool
	}{
		{
			name: "has added items",
			diff: &ScheduleDiff{
				Added: Schedule{"key1": {}},
			},
			expected: true,
		},
		{
			name: "has deleted items",
			diff: &ScheduleDiff{
				Deleted: Schedule{"key1": {}},
			},
			expected: true,
		},
		{
			name: "has modified items",
			diff: &ScheduleDiff{
				Modified: Schedule{"key1": {}},
			},
			expected: true,
		},
		{
			name: "no changes",
			diff: &ScheduleDiff{
				Added:     Schedule{},
				Deleted:   Schedule{},
				Modified:  Schedule{},
				Unchanged: Schedule{"key1": {}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.diff.HasChanges()
			if result != tt.expected {
				t.Errorf("HasChanges() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSerializeDeserializeSchedule(t *testing.T) {
	original := Schedule{
		"MONDAY, 12/5": {
			DayOfWeek:  "MONDAY",
			DayOfMonth: "12/5",
			Location:   "Field A",
			TimeBlock:  "3:00pm",
			ParsedTime: &time.Time{}, // Note: time serialization might be tricky
		},
	}

	// Test serialization
	data, err := SerializeSchedule(original)
	if err != nil {
		t.Fatalf("SerializeSchedule() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("SerializeSchedule() returned empty data")
	}

	// Verify it's valid JSON
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		t.Errorf("SerializeSchedule() produced invalid JSON: %v", err)
	}

	// Test deserialization
	result, err := DeserializeSchedule(data)
	if err != nil {
		t.Fatalf("DeserializeSchedule() error = %v", err)
	}

	if len(result) != len(original) {
		t.Errorf("DeserializeSchedule() count = %d, want %d", len(result), len(original))
	}

	// Check key fields (excluding ParsedTime for simplicity)
	for key, originalEntry := range original {
		resultEntry, exists := result[key]
		if !exists {
			t.Errorf("DeserializeSchedule() missing key: %s", key)
			continue
		}

		if resultEntry.DayOfWeek != originalEntry.DayOfWeek {
			t.Errorf("DeserializeSchedule() key %s DayOfWeek = %s, want %s", key, resultEntry.DayOfWeek, originalEntry.DayOfWeek)
		}
		if resultEntry.Location != originalEntry.Location {
			t.Errorf("DeserializeSchedule() key %s Location = %s, want %s", key, resultEntry.Location, originalEntry.Location)
		}
	}
}

func TestDeserializeSchedule_EmptyData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil data", nil},
		{"empty data", []byte{}},
		{"empty JSON", []byte("{}")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeSchedule(tt.data)
			if err != nil {
				t.Errorf("DeserializeSchedule() error = %v", err)
			}
			if result == nil {
				t.Error("DeserializeSchedule() returned nil")
			}
			if len(result) != 0 {
				t.Errorf("DeserializeSchedule() count = %d, want 0", len(result))
			}
		})
	}
}

func TestGetTimestampedFilename(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		extension string
		wantBase  string
		wantExt   string
	}{
		{
			name:      "default values",
			base:      "schedule-screenshot",
			extension: "png",
			wantBase:  "schedule-screenshot",
			wantExt:   "png",
		},
		{
			name:      "custom values",
			base:      "custom-file",
			extension: "json",
			wantBase:  "custom-file",
			wantExt:   "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTimestampedFilename(tt.base, tt.extension)

			if result == "" {
				t.Error("GetTimestampedFilename() returned empty string")
			}

			// Check that it contains the base name
			if !contains(result, tt.wantBase) {
				t.Errorf("GetTimestampedFilename() = %s, should contain %s", result, tt.wantBase)
			}

			// Check that it ends with the extension
			if !endsWith(result, "."+tt.wantExt) {
				t.Errorf("GetTimestampedFilename() = %s, should end with .%s", result, tt.wantExt)
			}

			// Check that it contains year (basic timestamp validation)
			currentYear := time.Now().Year()
			yearStr := string(rune(currentYear/1000+'0')) + string(rune((currentYear/100)%10+'0')) + string(rune((currentYear/10)%10+'0')) + string(rune(currentYear%10+'0'))
			if !contains(result, yearStr) {
				t.Errorf("GetTimestampedFilename() = %s, should contain current year %s", result, yearStr)
			}
		})
	}
}

// Integration test with real scraped content
func TestParseSchedule_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Import the scraper here to avoid circular dependencies in main tests
	// In real usage, this would come from the scraper module
	t.Run("integration with scraper", func(t *testing.T) {
		// This is a placeholder for integration testing with the actual scraper
		// In practice, you would:
		// 1. Use the scraper to get real HTML content
		// 2. Parse it with ParseSchedule
		// 3. Verify the results make sense

		// For now, test with realistic HTML content based on what we saw
		realisticHTML := `<span style="color:#0B1C2F;" class="wixui-rich-text__text">Our plan for the week ahead.</span>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">SATURDAY, 9/6</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Team Practice, Warren, 3:30–6:00</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:#184E99;" class="wixui-rich-text__text">TUESDAY, 9/9</span></p>
<p class="font_8 wixui-rich-text__text" style="line-height:1.2em; font-size:12px;">
<span style="color:rgb(11, 28, 47);" class="wixui-rich-text__text">Team Practice, Warren, 3:30–6:00</span></p>`

		result, err := ParseSchedule(realisticHTML)
		if err != nil {
			t.Fatalf("ParseSchedule() error = %v", err)
		}

		// Verify we got some results
		if len(result) == 0 {
			t.Error("ParseSchedule() returned empty schedule")
		}

		// Verify the structure
		for key, entry := range result {
			if entry.DayOfWeek == "" {
				t.Errorf("Entry %s has empty DayOfWeek", key)
			}
			if entry.DayOfMonth == "" {
				t.Errorf("Entry %s has empty DayOfMonth", key)
			}
			// Location or TimeBlock should be present
			if entry.Location == "" && entry.TimeBlock == "" {
				t.Errorf("Entry %s has both empty Location and TimeBlock", key)
			}

			t.Logf("Parsed entry %s: %+v", key, entry)
		}
	})
}

// Integration test with mock S3 (demonstrates the pattern)
func TestSaveLoadSchedule_Mock(t *testing.T) {
	// This would require a mock S3 client
	t.Skip("Would require mock S3 client implementation")

	// Expected pattern:
	// 1. Create a mock S3 client that implements the same interface
	// 2. Test SaveSchedule and LoadSchedule with the mock
	// 3. Verify the correct data is passed to S3 operations
}

// Helper functions for string operations
func contains(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Benchmark tests
func BenchmarkParseSchedule(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseSchedule(sampleScheduleHTML)
		if err != nil {
			b.Errorf("ParseSchedule() error = %v", err)
		}
	}
}

func BenchmarkParseScheduleText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseSchedule(sampleScheduleText)
		if err != nil {
			b.Errorf("ParseSchedule() error = %v", err)
		}
	}
}

func BenchmarkCompareSchedules(b *testing.B) {
	schedule1 := Schedule{
		"MONDAY, 12/5":  {DayOfWeek: "MONDAY", DayOfMonth: "12/5", Location: "Field A"},
		"TUESDAY, 12/6": {DayOfWeek: "TUESDAY", DayOfMonth: "12/6", Location: "Field B"},
	}
	schedule2 := Schedule{
		"MONDAY, 12/5":    {DayOfWeek: "MONDAY", DayOfMonth: "12/5", Location: "Field C"},
		"WEDNESDAY, 12/7": {DayOfWeek: "WEDNESDAY", DayOfMonth: "12/7", Location: "Field D"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CompareSchedules(schedule1, schedule2)
	}
}
