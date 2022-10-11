package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	var input string

	if len(os.Args) > 1 {
		input = os.Args[1]
		fmt.Println("Migration name is " + input)
	} else {
		fmt.Println("Migration name:")
		fmt.Scanln(&input)
	}
	stamp := time.Now().In(time.UTC).Format("20060102150405")
	for _, direction := range []string{"up", "down"} {
		filename := fmt.Sprintf("%s_%s.%s.sql", stamp, input, direction)
		file, err := os.Create("./migrations/" + filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
	}
}
