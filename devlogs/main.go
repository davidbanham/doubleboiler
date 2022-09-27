package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	trace := []string{}
	lastLineWasFilePos := false
	lastLineWasEOF := false

	compilationErrorsExist := false

	var line string
	var err error

	for {
		line, err = reader.ReadString('\n')
		if lastLineWasEOF && err == nil {
			lastLineWasEOF = false
		}
		if err != nil && err != io.EOF {
			break
		}

		line = strings.ReplaceAll(line, "\n", "")

		if strings.Contains(line, "panic") {
			fmt.Println(line)
		}

		if strings.Contains(line, "DEBUG") {
			if !strings.Contains(line, "kewpie") {
				fmt.Println(line)
			}
		}

		if strings.Contains(line, "Sending Error Response") {
			fmt.Println(line)
		}

		if strings.Contains(line, "Running build command!") {
			fmt.Println(line)
		}

		if strings.Contains(line, "Build ok.") {
			fmt.Println(line)
			compilationErrorsExist = false
		}

		if strings.Contains(line, "Error while building") {
			compilationErrorsExist = true
		}

		if compilationErrorsExist {
			fmt.Println(line)
		}

		if strings.Contains(line, "goroutine") {
			trace = append(trace, line)
			lastLineWasFilePos = true
		} else if lastLineWasFilePos {
			if strings.Contains(line, "bestwork") {
				trace = append(trace, line)
			}
			lastLineWasFilePos = false
		} else if strings.Contains(line, ": 	/") {
			if strings.Contains(line, "bestwork") {
				trace = append(trace, line)
			}
			lastLineWasFilePos = true
		} else if len(trace) != 0 {
			if len(trace) > 10 {
				fmt.Println(strings.Join(trace[:11], "\n"))
			} else if len(trace) == 1 {
				bits := strings.Split(trace[0], " 	")
				fmt.Println(strings.Join(bits, "\n"))
			} else {
				fmt.Println(strings.Join(trace, "\n"))
			}
			trace = []string{}
			lastLineWasFilePos = false
		}

		if err != nil {
			if !lastLineWasEOF {
				fmt.Println("EOF hit, an upstream process has exited.")
			}
			lastLineWasEOF = true
			time.Sleep(2 * time.Second)
		}
	}
	if err != io.EOF {
		log.Fatal(err)
	}
}
