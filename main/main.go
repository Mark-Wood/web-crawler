package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func main() {
	var err error
	maxDepth := -1

	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Println(errors.New("Usage: web-crawler <url> [max-depth]"))
		os.Exit(1)
	}

	if len(os.Args) == 3 {
		maxDepth, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println(err)
			fmt.Println(errors.New("Usage: web-crawler <url> [max-depth]"))
			os.Exit(1)
		}
	}
	page, err := Crawl(os.Args[1], maxDepth)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Print(page)
}
