package changelog

import (
	"sort"
	"time"
)

func init() {
	items := []Item{
		Item{
			Title: "Hello World",
			Body:  `Welcome to Double Boiler`,
		},
	}

	now, err := time.Parse(time.RFC3339Nano, "2020-02-10T01:18:18+00:00")

	if err != nil {
		return
	}

	Changes = append(Changes, Change{
		Date:  now,
		Items: items,
	})

	sort.Sort(Changes)
}
