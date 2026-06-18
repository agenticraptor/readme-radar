package llm

import (
	"context"
	"errors"
	"os"
	"strings"
)

// DefaultOpenAIModel is the default chat model. Override via config, --model,
// or README_RADAR_MODEL.
const DefaultOpenAIModel = "gpt-4o-mini"

type openAIClient struct {
	apiKey  string
	model   string
	baseURL string
}

func newOpenAI(model, baseURL string) (Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, ErrNoCredentials
	}
	if model == "" {
		model = DefaultOpenAIModel
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &openAIClient{apiKey: key, model: model, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (c *openAIClient) Name() string  { return "openai" }
func (c *openAIClient) Model() string { return c.model }

func (c *openAIClient) Complete(ctx context.Context, req Request) (string, error) {
	messages := []map[string]string{}
	if req.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.Prompt})

	payload := map[string]any{
		"model":       c.model,
		"messages":    messages,
		"temperature": req.Temperature,
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := postJSON(ctx, c.baseURL+"/v1/chat/completions", headers, payload, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", errors.New(out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openai: empty response")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
