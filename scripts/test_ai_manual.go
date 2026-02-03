package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gnemet/SlideForge/internal/ai"
	"github.com/gnemet/SlideForge/internal/config"
)

func main() {
	// 1. Manually setup a config with a mock provider for testing
	cfg := &config.Config{
		AI: config.AIConfig{
			ActiveProvider: "dev_mock",
			Providers: map[string]config.ProviderSettings{
				"dev_mock": {
					Driver: "mock",
				},
			},
		},
	}

	// 2. Initialize the AI Client
	client := ai.NewClient(cfg)

	// 3. Define the text to summarize
	slideText := "The SlideForge application uses a Library-First architecture to ensure high performance and reusability. It integrates with Google Drive for storage and uses HTMX for a modern, responsive user interface."

	fmt.Printf("Input Text: %s\n\n", slideText)

	// 4. Call SummarizeText
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := client.SummarizeText(ctx, slideText)
	if err != nil {
		log.Fatalf("AI Error: %v", err)
	}

	// 5. Output the result
	fmt.Printf("AI Summary: %s\n", summary)
}
