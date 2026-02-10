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

// Persona defines the behavioral and technical rules for the AI.
type Persona struct {
	Name        string
	Role        string
	Instruction string
}

// PresentationArchitect is the primary persona for SlideForge.
var PresentationArchitect = Persona{
	Name:        "Presentation Architect",
	Role:        "You are the 'Presentation Architect' for SlideForge, a high-performance PPTX foundry. You are an expert in slide structure, visual hierarchy, and information density.",
	Instruction: "Respond concisely and professionally. Focus on information fidelity and modern presentation standards. When extracting titles or summaries, provide the content directly without preamble.",
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Result struct {
	Content string  `json:"content"`
	Usage   Usage   `json:"usage"`
	Cost    float64 `json:"cost"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{Config: cfg}
}

func (c *Client) AnalyzeSlide(ctx context.Context, pngPath string) (Result, error) {
	prompt := fmt.Sprintf("Analyze this slide thumbnail: %s. Describe the main topic, key points, and visual style.", pngPath)
	return c.GenerateContent(ctx, PresentationArchitect, prompt)
}

func (c *Client) GenerateContent(ctx context.Context, persona Persona, prompt string) (Result, error) {
	activeName := c.Config.AI.ActiveProvider
	settings, ok := c.Config.AI.Providers[activeName]
	if !ok {
		return Result{}, fmt.Errorf("AI provider %s not configured", activeName)
	}

	switch settings.Driver {
	case "gemini":
		return c.generateWithGemini(ctx, settings, persona, prompt)
	case "openai":
		return c.generateWithOpenAICompatible(ctx, settings, persona, prompt)
	case "anthropic":
		return c.generateWithClaude(ctx, settings, persona, prompt)
	case "mock":
		return c.generateWithMock(ctx, settings, persona, prompt)
	default:
		return Result{}, fmt.Errorf("unsupported AI driver: %s", settings.Driver)
	}
}

func (c *Client) SummarizeText(ctx context.Context, text string) (Result, error) {
	prompt := fmt.Sprintf("Summarize the following text from a presentation slide in one or two concise sentences: %s", text)
	return c.GenerateContent(ctx, PresentationArchitect, prompt)
}

func (c *Client) ExtractTitle(ctx context.Context, firstSlideText string) (Result, error) {
	prompt := fmt.Sprintf("Based on the following text from the first slide of a presentation, extract the main title of the deck: %s", firstSlideText)
	return c.GenerateContent(ctx, PresentationArchitect, prompt)
}

func (c *Client) ExtractSlideTitle(ctx context.Context, slideText string) (Result, error) {
	prompt := fmt.Sprintf("Generate a very short title (max 5 words) for this slide content: %s", slideText)
	return c.GenerateContent(ctx, PresentationArchitect, prompt)
}

func (c *Client) ExtractTitleFromComments(ctx context.Context, comments string) (Result, error) {
	prompt := fmt.Sprintf("Generate a very short title (max 5 words) for a slide based on these user comments: %s", comments)
	return c.GenerateContent(ctx, PresentationArchitect, prompt)
}

func (c *Client) calculateCost(usage Usage, p config.ProviderSettings) float64 {
	if !p.IsPaid {
		return 0
	}
	inputCost := (float64(usage.PromptTokens) / 1000000.0) * p.InputPricePer1M
	outputCost := (float64(usage.CompletionTokens) / 1000000.0) * p.OutputPricePer1M
	return inputCost + outputCost
}

func (c *Client) generateWithMock(ctx context.Context, s config.ProviderSettings, persona Persona, prompt string) (Result, error) {
	// Simple mock for testing without API calls
	return Result{
		Content: fmt.Sprintf("[MOCK SUMMARY for: %s...]", prompt[:min(len(prompt), 30)]),
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Cost: 0,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) generateWithGemini(ctx context.Context, s config.ProviderSettings, persona Persona, prompt string) (Result, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(s.Key))
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	modelName := s.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	model := client.GenerativeModel(modelName)

	// Implement SystemInstruction Pattern from jiramntr
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(persona.Role + "\n" + persona.Instruction)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return Result{}, err
	}

	usage := Usage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.CompletionTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		usage.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return Result{
			Content: fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]),
			Usage:   usage,
			Cost:    c.calculateCost(usage, s),
		}, nil
	}
	return Result{}, fmt.Errorf("no response from Gemini")
}

func (c *Client) generateWithOpenAICompatible(ctx context.Context, s config.ProviderSettings, persona Persona, prompt string) (Result, error) {
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
		Model: s.Model,
		Messages: []Message{
			{Role: "system", Content: persona.Role + "\n" + persona.Instruction},
			{Role: "user", Content: prompt},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Key)

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_token_count"` // This varies by API, standard is different
			CompletionTokens int `json:"completion_token_count"`
			TotalTokens      int `json:"total_token_count"`
		} `json:"usage"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	usage := Usage{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
	}

	if len(result.Choices) > 0 {
		return Result{
			Content: result.Choices[0].Message.Content,
			Usage:   usage,
			Cost:    c.calculateCost(usage, s),
		}, nil
	}
	return Result{}, fmt.Errorf("no response from OpenAI")
}

func (c *Client) generateWithClaude(ctx context.Context, s config.ProviderSettings, persona Persona, prompt string) (Result, error) {
	// Minimal Claude implementation
	type Request struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		System    string `json:"system"`
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
		System:    persona.Role + "\n" + persona.Instruction,
	}
	reqBody.Messages = append(reqBody.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: prompt})

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	usage := Usage{
		PromptTokens:     result.Usage.InputTokens,
		CompletionTokens: result.Usage.OutputTokens,
		TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
	}

	if len(result.Content) > 0 {
		return Result{
			Content: result.Content[0].Text,
			Usage:   usage,
			Cost:    c.calculateCost(usage, s),
		}, nil
	}
	return Result{}, fmt.Errorf("no response from Claude")
}
