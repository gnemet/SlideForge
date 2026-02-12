package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gnemet/SlideForge/internal/pptx"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/extract_context.go <pptx_path>")
	}
	path := os.Args[1]
	content, err := pptx.ExtractSlideContent(path)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := json.MarshalIndent(content, "", "  ")
	fmt.Println(string(data))
}
