package util

import "fmt"

func Dollarise(in int) string {
	return fmt.Sprintf("$%.2f", float64(in)/float64(100))
}
