package changelog

import (
	"sort"
	"time"
)

func init() {
	now, err := time.Parse(time.RFC3339Nano, "{{now}}")

	if err != nil {
		return
	}

	Changes = append(Changes, Change{
		Date:  now,
		Title: "",
		Body:  ``,
	})

	sort.Sort(Changes)
}
