package util

func Uniq(input []string) []string {
	seen := map[string]bool{}
	ret := []string{}

	for _, str := range input {
		if seen[str] {
			continue
		}
		seen[str] = true
		ret = append(ret, str)
	}

	return ret
}
