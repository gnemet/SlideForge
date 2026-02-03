package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gnemet/SlideForge/internal/config"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Client struct {
	Config *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{Config: cfg}
}

func (c *Client) AnalyzeSlide(ctx context.Context, pngPath string) (string, error) {
	prompt := fmt.Sprintf("Analyze this slide thumbnail: %s. Describe the main topic, key points, and visual style.", pngPath)
	return c.GenerateContent(ctx, prompt)
}

func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	activeName := c.Config.AI.ActiveProvider
	settings, ok := c.Config.AI.Providers[activeName]
	if !ok {
		return "", fmt.Errorf("AI provider %s not configured", activeName)
	}

	switch settings.Driver {
	case "gemini":
		return c.generateWithGemini(ctx, settings, prompt)
	case "openai":
		return c.generateWithOpenAICompatible(ctx, settings, prompt)
	case "anthropic":
		return c.generateWithClaude(ctx, settings, prompt)
	case "mock":
		return c.generateWithMock(ctx, settings, prompt)
	default:
		return "", fmt.Errorf("unsupported AI driver: %s", settings.Driver)
	}
}

func (c *Client) SummarizeText(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf("Summarize the following text from a presentation slide in one or two concise sentences: %s", text)
	return c.GenerateContent(ctx, prompt)
}

func (c *Client) ExtractTitle(ctx context.Context, firstSlideText string) (string, error) {
	prompt := fmt.Sprintf("Based on the following text from the first slide of a presentation, extract the main title of the deck. Respond with ONLY the title itself, no punctuation or extra words: %s", firstSlideText)
	return c.GenerateContent(ctx, prompt)
}

func (c *Client) ExtractSlideTitle(ctx context.Context, slideText string) (string, error) {
	prompt := fmt.Sprintf("Generate a very short title (max 5 words) for this slide content. Return ONLY the title: %s", slideText)
	return c.GenerateContent(ctx, prompt)
}

func (c *Client) generateWithMock(ctx context.Context, s config.ProviderSettings, prompt string) (string, error) {
	// Simple mock for testing without API calls
	return fmt.Sprintf("[MOCK SUMMARY for: %s...]", prompt[:min(len(prompt), 30)]), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) generateWithGemini(ctx context.Context, s config.ProviderSettings, prompt string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(s.Key))
	if err != nil {
		return "", err
	}
	defer client.Close()

	modelName := s.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	model := client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "", fmt.Errorf("no response from Gemini")
}

func (c *Client) generateWithOpenAICompatible(ctx context.Context, s config.ProviderSettings, prompt string) (string, error) {
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type Request struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
	}

	reqBody := Request{
		Model:    s.Model,
		Messages: []Message{{Role: "user", Content: prompt}},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Key)

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response from OpenAI")
}

func (c *Client) generateWithClaude(ctx context.Context, s config.ProviderSettings, prompt string) (string, error) {
	// Minimal Claude implementation
	type Request struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	maxTokens := s.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	reqBody := Request{
		Model:     s.Model,
		MaxTokens: maxTokens,
	}
	reqBody.Messages = append(reqBody.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: prompt})

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("no response from Claude")
}
