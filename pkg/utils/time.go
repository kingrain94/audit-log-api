package utils

import (
	"fmt"
	"time"
)

// ParseUserTime parses a time string that can be either RFC3339 or YYYY-MM-DD format.
// For YYYY-MM-DD format, if isEndTime is true, it will set the time to end of day (23:59:59).
func ParseUserTime(timeStr string, isEndTime bool) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try simple date format
	t, err = time.Parse("2006-01-02", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, expected RFC3339 or YYYY-MM-DD, got %s", timeStr)
	}

	// For end_time with date only, set it to end of day
	if isEndTime {
		t = t.Add(24*time.Hour - time.Second)
	}

	return t, nil
}
