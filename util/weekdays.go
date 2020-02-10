package util

import "time"

func IsWeekday(day time.Time) bool {
	return !IsWeekend(day)
}

func IsWeekend(day time.Time) bool {
	wd := day.Weekday()
	if wd == time.Friday || wd == time.Saturday {
		return true
	}
	return false
}
