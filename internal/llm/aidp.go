package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Config struct {
	URL       string
	AK        string
	Model     string
	MaxTokens int
	Stream    bool
}

type AIDPClient struct {
	cfg        Config
	httpClient *http.Client
}

func NewAIDPClient(cfg Config, httpClient *http.Client) *AIDPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &AIDPClient{cfg: cfg, httpClient: httpClient}
}

func (c *AIDPClient) Generate(ctx context.Context, prompt string) (string, error) {
	endpoint, err := url.Parse(c.cfg.URL)
	if err != nil {
		return "", fmt.Errorf("invalid AIDP URL")
	}
	query := endpoint.Query()
	query.Set("ak", c.cfg.AK)
	endpoint.RawQuery = query.Encode()

	body := map[string]any{
		"stream":     c.cfg.Stream,
		"model":      c.cfg.Model,
		"max_tokens": c.cfg.MaxTokens,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("AIDP request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("AIDP request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("AIDP response did not include report content")
	}
	return result.Choices[0].Message.Content, nil
}
