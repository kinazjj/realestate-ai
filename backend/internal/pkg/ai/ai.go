package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Provider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

type GeminiClient struct {
	APIKey string
	Model  string
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	return &GeminiClient{APIKey: apiKey, Model: model}
}

func (c *GeminiClient) Chat(ctx context.Context, messages []Message) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.Model, c.APIKey)

	contents := make([]geminiContent, 0, len(messages))
	for _, m := range messages {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	body := geminiRequest{Contents: contents}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: %s", resp.Status)
	}

	var gemResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gemResp); err != nil {
		return "", err
	}

	if gemResp.Error != nil {
		return "", fmt.Errorf("gemini error: %s", gemResp.Error.Message)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty gemini response")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

type GroqClient struct {
	APIKey string
	Model  string
}

type groqRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"error_message"`
	} `json:"error,omitempty"`
}

func NewGroqClient(apiKey, model string) *GroqClient {
	return &GroqClient{APIKey: apiKey, Model: model}
}

func (c *GroqClient) Chat(ctx context.Context, messages []Message) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	body := groqRequest{
		Model:    c.Model,
		Messages: messages,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API error: status %d", resp.StatusCode)
	}

	var groqResp groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", err
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("groq error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("empty groq response")
	}

	return groqResp.Choices[0].Message.Content, nil
}

type Router struct {
	providers []Provider
}

func NewRouterFromEnv() *Router {
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3.1-flash-lite-preview"
	}

	geminiKeysRaw := os.Getenv("GEMINI_KEYS")
	var geminiKeys []string
	if geminiKeysRaw != "" {
		geminiKeysRaw = strings.Trim(geminiKeysRaw, "[]")
		geminiKeys = parseJSONArray(geminiKeysRaw)
	}

	groqKeysRaw := os.Getenv("GROQ_KEYS")
	var groqKeys []string
	if groqKeysRaw != "" {
		groqKeysRaw = strings.Trim(groqKeysRaw, "[]")
		groqKeys = parseJSONArray(groqKeysRaw)
	}

	groqModelsRaw := os.Getenv("GROQ_MODELS")
	var groqModels []string
	if groqModelsRaw != "" {
		groqModelsRaw = strings.Trim(groqModelsRaw, "[]")
		groqModels = parseJSONArray(groqModelsRaw)
	}

	providers := []Provider{}

	for _, key := range geminiKeys {
		providers = append(providers, NewGeminiClient(key, geminiModel))
	}

	for i, key := range groqKeys {
		model := "llama-3.3-70b-versatile"
		if i < len(groqModels) {
			model = groqModels[i]
		}
		providers = append(providers, NewGroqClient(key, model))
	}

	return &Router{providers: providers}
}

func parseJSONArray(raw string) []string {
	var result []string
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (r *Router) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(r.providers) == 0 {
		return "", fmt.Errorf("no AI providers configured")
	}

	var lastErr error
	for _, provider := range r.providers {
		resp, err := provider.Chat(ctx, messages)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}
