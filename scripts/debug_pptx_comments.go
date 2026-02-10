package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/gnemet/SlideForge/internal/pptx"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/debug_pptx_comments.go <path_to_pptx>")
	}
	path := os.Args[1]

	content, err := pptx.ExtractSlideContent(path)
	if err != nil {
		log.Fatalf("Failed to extract content: %v", err)
	}

	fmt.Printf("Extracted data for %d slides\n", len(content))

	// Get slide numbers and sort them
	var slideNums []int
	for n := range content {
		slideNums = append(slideNums, n)
	}
	sort.Ints(slideNums)

	for _, n := range slideNums {
		sl := content[n]
		if len(sl.Comments) > 0 {
			fmt.Printf("\n--- Slide %d ---\n", sl.SlideNumber)
			for j, c := range sl.Comments {
				fmt.Printf("Comment %d [%s]: %s\n", j, c.Author, c.Text)
			}
		}
	}
}
