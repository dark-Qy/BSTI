package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	return c.GenerateWithValidation(ctx, prompt, nil)
}

func (c *AIDPClient) GenerateWithValidation(ctx context.Context, prompt string, validate func(string) error) (string, error) {
	maxTokens := startingMaxTokens(c.cfg.MaxTokens)
	requestPrompt := prompt
	validationAttempts := 0
	for {
		body, err := c.buildRequestBody(requestPrompt, maxTokens)
		if err != nil {
			return "", err
		}
		raw, err := c.doRequest(ctx, body)
		if err != nil {
			return "", err
		}
		content, ok := extractReportContent(raw)
		if ok {
			if validate == nil {
				return content, nil
			}
			if err := validate(content); err == nil {
				return content, nil
			} else if validationAttempts == 0 {
				validationAttempts++
				requestPrompt = prompt + "\n\n你上一次的输出没有通过校验，原因是：" + err.Error() + "。请重新输出一次，只输出合法 JSON。"
				continue
			} else {
				return "", fmt.Errorf("AIDP response validation failed: %w", err)
			}
		}
		if !shouldRetryEmptyLength(raw) {
			return "", fmt.Errorf("AIDP response did not include report content (shape: %s)", aidpResponseShape(raw))
		}
		next := retryMaxTokens(maxTokens)
		if next <= maxTokens {
			return "", fmt.Errorf("AIDP response did not include report content (shape: %s)", aidpResponseShape(raw))
		}
		maxTokens = next
	}
}

func (c *AIDPClient) buildRequestBody(prompt string, maxTokens int) (map[string]any, error) {
	endpoint, err := url.Parse(c.cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid AIDP URL")
	}
	_ = endpoint
	return map[string]any{
		"stream":     c.cfg.Stream,
		"model":      c.cfg.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
	}, nil
}

func (c *AIDPClient) doRequest(ctx context.Context, body map[string]any) (map[string]json.RawMessage, error) {
	endpoint, err := url.Parse(c.cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid AIDP URL")
	}
	query := endpoint.Query()
	query.Set("ak", c.cfg.AK)
	endpoint.RawQuery = query.Encode()

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AIDP request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AIDP request failed with status %d", resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type aidpChoice struct {
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type aidpRawChoice struct {
	FinishReason string                     `json:"finish_reason"`
	Message      map[string]json.RawMessage `json:"message"`
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

	choicesLen := 0
	messageKeys := []string(nil)
	contentType := "missing"
	contentLen := -1
	finishReason := ""
	var result struct {
		Choices []aidpRawChoice `json:"choices"`
	}
	if err := unmarshalRaw(raw, &result); err == nil {
		choicesLen = len(result.Choices)
		if choicesLen > 0 {
			choice := result.Choices[0]
			finishReason = choice.FinishReason
			for key := range choice.Message {
				messageKeys = append(messageKeys, key)
			}
			sort.Strings(messageKeys)
			content := choice.Message["content"]
			contentType = jsonType(content)
			contentLen = jsonStringLen(content)
		}
	}
	return fmt.Sprintf("top_level_keys=%s choices_len=%d message_keys=%s content_type=%s content_len=%d finish_reason=%s", strings.Join(keys, ","), choicesLen, strings.Join(messageKeys, ","), contentType, contentLen, finishReason)
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

func jsonStringLen(raw json.RawMessage) int {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return -1
	}
	return len(text)
}

func shouldRetryEmptyLength(raw map[string]json.RawMessage) bool {
	var result struct {
		Choices []aidpRawChoice `json:"choices"`
	}
	if err := unmarshalRaw(raw, &result); err != nil || len(result.Choices) == 0 {
		return false
	}
	choice := result.Choices[0]
	content := choice.Message["content"]
	return choice.FinishReason == "length" && jsonType(content) == "string" && jsonStringLen(content) == 0
}

func startingMaxTokens(current int) int {
	if current < 5000 {
		return 5000
	}
	return current
}

func retryMaxTokens(current int) int {
	if current > math.MaxInt/2 {
		return current
	}
	return current * 2
}
