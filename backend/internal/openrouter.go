package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ORClient struct {
	HTTP *http.Client
}

func NewORClient() *ORClient {
	return &ORClient{HTTP: &http.Client{Timeout: 120 * time.Second}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type reasoningCfg struct {
	Effort string `json:"effort,omitempty"`
}

type providerCfg struct {
	Only           []string `json:"only,omitempty"`
	AllowFallbacks *bool    `json:"allow_fallbacks,omitempty"`
}

// buildProvider parses a comma/space separated list of OpenRouter provider
// slugs (e.g. "openai, anthropic") into a provider routing object that
// whitelists exactly those providers. Returns nil when the list is empty
// so the field is omitted and OpenRouter uses its default routing.
func buildProvider(only string) *providerCfg {
	only = strings.TrimSpace(only)
	if only == "" {
		return nil
	}
	parts := strings.FieldsFunc(only, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	slugs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			slugs = append(slugs, p)
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	return &providerCfg{Only: slugs}
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
	Temperature    float64        `json:"temperature,omitempty"`
	Reasoning      *reasoningCfg  `json:"reasoning,omitempty"`
	Provider       *providerCfg   `json:"provider,omitempty"`
	Stream         bool           `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat calls the chat-completions endpoint. effort is an OpenRouter
// reasoning level ("minimal"/"low"/"medium"/"high"/"xhigh"/"none"); empty
// string means "don't send a reasoning field" (use the model default).
func (c *ORClient) Chat(ctx context.Context, s Settings, model, prompt, effort string) (string, error) {
	if s.APIKey == "" {
		return "", errors.New("api key not configured")
	}
	body := chatRequest{
		Model:          model,
		Messages:       []chatMessage{{Role: "user", Content: prompt}},
		ResponseFormat: map[string]any{"type": "json_object"},
		Temperature:    0.3,
		Stream:         false,
	}
	if effort != "" {
		body.Reasoning = &reasoningCfg{Effort: effort}
	}
	// provider whitelist applies to chat tasks only (simplify + analyze);
	// TTS deliberately stays unrestricted.
	body.Provider = buildProvider(s.ProviderOnly)
	buf, _ := json.Marshal(body)
	url := strings.TrimRight(s.BaseURL, "/") + "/chat/completions"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/reader")
	req.Header.Set("X-Title", "Reader")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("upstream %d: %s", resp.StatusCode, string(raw))
	}
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", fmt.Errorf("decode: %w; body=%s", err, string(raw))
	}
	if cr.Error != nil {
		return "", fmt.Errorf("upstream: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", errors.New("no choices in response")
	}
	return cr.Choices[0].Message.Content, nil
}

type ttsRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format,omitempty"`
	Instructions   string `json:"instructions,omitempty"`
}

func (c *ORClient) TTS(ctx context.Context, s Settings, text string) ([]byte, string, error) {
	if s.APIKey == "" {
		return nil, "", errors.New("api key not configured")
	}
	body := ttsRequest{
		Model:          s.TTSModel,
		Input:          text,
		Voice:          s.Voice,
		ResponseFormat: "mp3",
		Instructions:   s.TTSInstruction,
	}
	buf, _ := json.Marshal(body)
	url := strings.TrimRight(s.BaseURL, "/") + "/audio/speech"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/reader")
	req.Header.Set("X-Title", "Reader")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("upstream %d: %s", resp.StatusCode, string(data))
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "audio/mpeg"
	}
	return data, ct, nil
}
