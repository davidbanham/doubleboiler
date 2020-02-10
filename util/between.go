package util

import "time"

func Between(target, start, end time.Time) bool {
	return target.After(start.Add(-time.Minute)) && target.Before(end)
}

func NightsBetween(start, end time.Time) []time.Time {
	allNights := []time.Time{}
	for d := start.Round(24 * time.Hour); d.Before(end); d = d.AddDate(0, 0, 1) {
		allNights = append(allNights, d)
	}
	return allNights
}
