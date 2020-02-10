package util

import "strings"

func FirstFiveChars(input string) string {
	if input == "" {
		return ""
	}
	return strings.ToUpper(input[:5])
}
