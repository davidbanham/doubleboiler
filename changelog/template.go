package changelog

import (
	"sort"
	"time"
)

func init() {
	items := []Item{
		Item{
			Title: "",
			Body:  ``,
		},
		Item{
			Title: "",
			Body:  ``,
		},
	}

	now, err := time.Parse(time.RFC3339Nano, "{{now}}")

	if err != nil {
		return
	}

	Changes = append(Changes, Change{
		Date:  now,
		Items: items,
	})

	sort.Sort(Changes)
}
