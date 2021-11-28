package util

import "strings"

func FirstFiveChars(input string) string {
	num := 5
	if len(input) < 5 {
		num = len(input)
	}
	return strings.ToUpper(input[:num])
}

func FirstChar(input string) string {
	if input == "" {
		return ""
	}
	return strings.ToUpper(input[:1])
}
