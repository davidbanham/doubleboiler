package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iancoleman/strcase"
)

func main() {
	textPtr := flag.String("text", "", "Text to parse.")
	formatPtr := flag.String("format", "camel", "Metric {camel|lower_camel|snake|kebab};.")
	flag.Parse()

	switch *formatPtr {
	default:
		fmt.Println("Invalid format. Try --help")
		os.Exit(1)
	case "camel":
		fmt.Println(strcase.ToCamel(*textPtr))
	case "lower_camel":
		fmt.Println(strcase.ToLowerCamel(*textPtr))
	case "snake":
		fmt.Println(strcase.ToSnake(*textPtr))
	case "kebab":
		fmt.Println(strcase.ToKebab(*textPtr))
	}
}
