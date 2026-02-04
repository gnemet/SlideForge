package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("GEMINI_KEY")
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash-preview-09-2025"
	}

	fmt.Printf("🛠️ Testing SlideForge Gemini Bridge...\n")
	fmt.Printf("📍 Model: %s\n", modelName)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text("Test connection. Reply with 'SlideForge Online'"))
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		fmt.Printf("✅ Success: %v\n", resp.Candidates[0].Content.Parts[0])
	}
}
