package util

func StripBlankStrings(input []string) []string {
	output := []string{}
	for _, str := range input {
		if str != "" {
			output = append(output, str)
		}
	}
	return output
}
