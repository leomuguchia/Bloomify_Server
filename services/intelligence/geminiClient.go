// File: service/ai/gemini_client.go
package ai

import (
	"context"
	"fmt"
	"strings"

	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	model *genai.GenerativeModel
}

func NewGeminiClient(apiKey string) *GeminiClient {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(fmt.Sprintf("failed to create Gemini client: %v", err))
	}

	model := client.GenerativeModel("models/gemini-1.5-pro")
	return &GeminiClient{model: model}
}

func (g *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate error: %w", err)
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			sb.WriteString(string(textPart))
		}
	}
	return sb.String(), nil
}
