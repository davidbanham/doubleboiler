package changelog

import "time"

type Change struct {
	Date  time.Time
	Items []Item
}

type Item struct {
	Title string
	Body  string
}

var Changes changeList

type changeList []Change

func (s changeList) Len() int {
	return len(s)
}
func (s changeList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s changeList) Less(i, j int) bool {
	return s[i].Date.After(s[j].Date)
}
