package util

func FirstNonEmptyString(inputs ...string) string {
	for _, input := range inputs {
		if input != "" {
			return input
		}
	}

	return ""
}
