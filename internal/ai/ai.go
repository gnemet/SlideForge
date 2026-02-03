package ai

import (
	"context"
	"fmt"
)

type Client struct {
	APIKey string
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

func (c *Client) AnalyzeSlide(ctx context.Context, pngPath string) (string, error) {
	// Mock AI analysis
	return fmt.Sprintf("AI Analysis for %s: This slide contains a strategic overview with 3 key pillars.", pngPath), nil
}

func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	return "Generated Content based on your prompt: [AI DATA]", nil
}
