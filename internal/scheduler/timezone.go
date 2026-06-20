package scheduler

import (
	"fmt"
	"time"
)

// TimezoneInfo holds information about a timezone.
type TimezoneInfo struct {
	Name     string
	Location *time.Location
	Offset   time.Duration
}

// GetTimezoneInfo returns information about the configured timezone.
func GetTimezoneInfo(tz string) (*TimezoneInfo, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tz, err)
	}

	_, offset := time.Now().In(loc).Zone()

	return &TimezoneInfo{
		Name:     tz,
		Location: loc,
		Offset:   time.Duration(offset) * time.Second,
	}, nil
}

// IsWIB checks if the timezone is Asia/Jakarta (WIB).
func IsWIB(tz string) bool {
	return tz == "Asia/Jakarta"
}

// FormatTimeInTZ formats a time in the given timezone.
func FormatTimeInTZ(t time.Time, tz string, format string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", fmt.Errorf("failed to load timezone %s: %w", tz, err)
	}

	return t.In(loc).Format(format), nil
}

// NowInTZ returns the current time in the given timezone.
func NowInTZ(tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load timezone %s: %w", tz, err)
	}

	return time.Now().In(loc), nil
}

// GetNextDeliveryTimes calculates the next delivery times for the day.
func GetNextDeliveryTimes(tz string, cronExpr string, count int) ([]time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tz, err)
	}

	now := time.Now().In(loc)
	var times []time.Time

	// Default delivery times: 00:00, 03:00, 06:00, 09:00, 12:00, 15:00, 18:00, 21:00
	deliveryHours := []int{0, 3, 6, 9, 12, 15, 18, 21}

	for _, hour := range deliveryHours {
		if len(times) >= count {
			break
		}

		deliveryTime := time.Date(
			now.Year(), now.Month(), now.Day(),
			hour, 0, 0, 0,
			loc,
		)

		if deliveryTime.After(now) {
			times = append(times, deliveryTime)
		}
	}

	// If no more deliveries today, get tomorrow's first delivery
	if len(times) == 0 && len(deliveryHours) > 0 {
		tomorrow := time.Date(
			now.Year(), now.Month(), now.Day()+1,
			deliveryHours[0], 0, 0, 0,
			loc,
		)
		times = append(times, tomorrow)
	}

	return times, nil
}

// IsActiveHour checks if the given hour is within active hours.
func IsActiveHour(hour, activeStart, activeEnd int) bool {
	if activeStart <= activeEnd {
		return hour >= activeStart && hour <= activeEnd
	}
	// Handle overnight (e.g., 22:00 - 06:00)
	return hour >= activeStart || hour <= activeEnd
}

// GetCurrentHourInTZ returns the current hour in the given timezone.
func GetCurrentHourInTZ(tz string) (int, error) {
	now, err := NowInTZ(tz)
	if err != nil {
		return 0, err
	}
	return now.Hour(), nil
}
