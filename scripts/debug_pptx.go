package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gnemet/SlideForge/internal/pptx"
)

func main() {
	pptxPath := "pptx/MINTA_B_ajánlat_DD_20250916_v5.pptx"
	if _, err := os.Stat(pptxPath); err != nil {
		log.Fatalf("File not found: %v", err)
	}

	content, err := pptx.ExtractSlideContent(pptxPath)
	if err != nil {
		log.Fatalf("Failed to extract content: %v", err)
	}

	for i := 1; i <= len(content); i++ {
		data := content[i]
		fmt.Printf("Slide %d:\n", i)
		fmt.Printf("  Text: %s\n", data.Text)
		fmt.Printf("  Comments: %d\n", len(data.Comments))
		for _, c := range data.Comments {
			fmt.Printf("    [%s]: %s\n", c.Author, c.Text)
		}
	}
}
