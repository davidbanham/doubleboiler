package util

import (
	"time"
)

func NextDay(begin time.Time, target time.Weekday) time.Time {
	if begin.Weekday() == target {
		return begin
	}
	return NextDay(begin.Add(24*time.Hour), target)
}
