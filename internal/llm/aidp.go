package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
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

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	content, ok := extractReportContent(raw)
	if !ok {
		return "", fmt.Errorf("AIDP response did not include report content (shape: %s)", aidpResponseShape(raw))
	}
	return content, nil
}

type aidpChoice struct {
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

func extractReportContent(raw map[string]json.RawMessage) (string, bool) {
	var result struct {
		Choices []aidpChoice `json:"choices"`
	}
	if err := unmarshalRaw(raw, &result); err != nil {
		return "", false
	}
	if len(result.Choices) == 0 || len(result.Choices[0].Message.Content) == 0 {
		return "", false
	}
	content := result.Choices[0].Message.Content
	var text string
	if err := json.Unmarshal(content, &text); err == nil && text != "" {
		return text, true
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &parts); err != nil {
		return "", false
	}
	var b strings.Builder
	for _, part := range parts {
		if part.Text != "" {
			b.WriteString(part.Text)
		}
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

func aidpResponseShape(raw map[string]json.RawMessage) string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var result struct {
		Choices []aidpChoice `json:"choices"`
	}
	choicesLen := 0
	messageKeys := []string(nil)
	contentType := "missing"
	if err := unmarshalRaw(raw, &result); err == nil {
		choicesLen = len(result.Choices)
		if choicesLen > 0 {
			var message map[string]json.RawMessage
			messageBytes, _ := json.Marshal(result.Choices[0].Message)
			if err := json.Unmarshal(messageBytes, &message); err == nil {
				for key := range message {
					messageKeys = append(messageKeys, key)
				}
				sort.Strings(messageKeys)
			}
			contentType = jsonType(result.Choices[0].Message.Content)
		}
	}
	return fmt.Sprintf("top_level_keys=%s choices_len=%d message_keys=%s content_type=%s", strings.Join(keys, ","), choicesLen, strings.Join(messageKeys, ","), contentType)
}

func unmarshalRaw(raw map[string]json.RawMessage, target any) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func jsonType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "missing"
	}
	switch raw[0] {
	case '"':
		return "string"
	case '[':
		return "array"
	case '{':
		return "object"
	case 'n':
		return "null"
	case 't', 'f':
		return "bool"
	default:
		return "number"
	}
}
