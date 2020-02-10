package util

import (
	"fmt"
	"time"
)

func DaysBetween(start, end time.Time) []time.Time {
	numDays := int(end.Sub(start).Hours() / 24)

	allDays := []time.Time{}

	for i := 0; i < numDays; i++ {
		days, _ := time.ParseDuration(fmt.Sprintf("%dh", i*24))
		allDays = append(allDays, start.Add(days))
	}

	return allDays
}
