package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gnemet/SlideForge/internal/ai"
	"github.com/gnemet/SlideForge/internal/config"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	fmt.Println("=== SlideForge AI Connection Tester ===")

	// 1. Load regular config (including .env)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	active := cfg.AI.ActiveProvider
	settings := cfg.AI.Providers[active]

	fmt.Printf("Active Provider: %s (Driver: %s)\n", active, settings.Driver)
	fmt.Printf("Model: %s\n", settings.Model)
	fmt.Printf("Configured Models: %d (%v)\n", len(settings.Models), settings.Models)

	if settings.Key == "" {
		fmt.Println("Warning: AI Key is EMPTY. Please set it in config.yaml or as an environment variable (e.g., GEMINI_KEY).")
	} else {
		maskedKey := settings.Key
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:4] + "..." + maskedKey[len(maskedKey)-4:]
		}
		fmt.Printf("API Key detected: %s\n", maskedKey)
	}

	// 2. Initialize the AI Client
	client := ai.NewClient(cfg)

	// 3. Test Prompt
	prompt := "Say 'The forge is hot!' if you are working correctly."

	fmt.Printf("\nSending test prompt: '%s'\n", prompt)
	fmt.Println("Waiting for response (timeout 30s)...")

	// 4. Call GenerateContent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if active == "gemini" {
		fmt.Println("\n--- Available Gemini Models ---")
		clientG, _ := genai.NewClient(ctx, option.WithAPIKey(settings.Key))
		iter := clientG.ListModels(ctx)
		for {
			m, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Printf("Error listing models: %v\n", err)
				break
			}
			fmt.Printf("- %s\n", m.Name)
		}
		clientG.Close()
		fmt.Println("-------------------------------\n")
	}

	start := time.Now()
	response, err := client.GenerateContent(ctx, prompt)
	if err != nil {
		fmt.Printf("\n❌ AI ERROR: %v\n", err)
		if active == "gemini" && err != nil {
			fmt.Println("\nTip: If you see a 403 'leaked' error, you MUST create a NEW API key in Google AI Studio")
			fmt.Println("and ensure it is NOT committed to GitHub. Use the .env file instead.")
		}
		return
	}

	// 5. Output the result
	fmt.Printf("\n✅ AI RESPONSE (%v):\n%s\n", time.Since(start).Round(time.Millisecond), response)
}
