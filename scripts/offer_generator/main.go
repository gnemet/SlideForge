package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gnemet/SlideForge/internal/pptx"
)

func main() {
	templatePath := flag.String("template", "", "Path to the sample PPTX template")
	outputPath := flag.String("output", "", "Path for the new generated PPTX")
	dataJSON := flag.String("data", "", "Metadata JSON string")
	dataFile := flag.String("data-file", "", "Path to metadata JSON file")
	flag.Parse()

	if *templatePath == "" || *outputPath == "" {
		fmt.Println("Usage: go run scripts/offer_generator/main.go -template <path> -output <path> [-data '<json>' | -data-file <path>]")
		os.Exit(1)
	}

	var data map[string]string
	if *dataJSON != "" {
		if err := json.Unmarshal([]byte(*dataJSON), &data); err != nil {
			log.Fatalf("Error parsing JSON data: %v", err)
		}
	} else if *dataFile != "" {
		bytes, err := os.ReadFile(*dataFile)
		if err != nil {
			log.Fatalf("Error reading data file: %v", err)
		}
		if err := json.Unmarshal(bytes, &data); err != nil {
			log.Fatalf("Error parsing JSON data: %v", err)
		}
	}

	if len(data) == 0 {
		log.Fatal("No metadata provided for placeholder replacement.")
	}

	fmt.Printf("Generating offer from %s...\n", *templatePath)
	if err := pptx.ReplacePlaceholders(*templatePath, *outputPath, data); err != nil {
		log.Fatalf("Error generating offer: %v", err)
	}

	fmt.Printf("Successfully generated new offer: %s\n", *outputPath)
}
