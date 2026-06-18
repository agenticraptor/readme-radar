package llm

import (
	"context"
	"errors"
	"os"
	"strings"
)

// DefaultOllamaModel is the default local model. Override via config, --model,
// or README_RADAR_MODEL. Requires a running Ollama (https://ollama.com).
const DefaultOllamaModel = "llama3.1"

type ollamaClient struct {
	model   string
	baseURL string
}

func newOllama(model, baseURL string) (Client, error) {
	if model == "" {
		model = DefaultOllamaModel
	}
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_HOST")
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "http://" + baseURL
	}
	return &ollamaClient{model: model, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (c *ollamaClient) Name() string  { return "ollama" }
func (c *ollamaClient) Model() string { return c.model }

func (c *ollamaClient) Complete(ctx context.Context, req Request) (string, error) {
	payload := map[string]any{
		"model":  c.model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.Prompt},
		},
		"options": map[string]any{"temperature": req.Temperature},
	}

	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := postJSON(ctx, c.baseURL+"/api/chat", nil, payload, &out); err != nil {
		return "", err
	}
	if out.Error != "" {
		return "", errors.New(out.Error)
	}
	return strings.TrimSpace(out.Message.Content), nil
}
