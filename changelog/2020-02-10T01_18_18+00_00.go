package changelog

import (
	"sort"
	"time"
)

func init() {
	now, err := time.Parse(time.RFC3339Nano, "2020-02-10T01:18:18+00:00")

	if err != nil {
		return
	}

	Changes = append(Changes, Change{
		Date:  now,
		Title: "Hello World",
		Body:  `Welcome to Double Boiler`,
	})

	sort.Sort(Changes)
}
