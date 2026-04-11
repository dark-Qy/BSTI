package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAIDPClientParsesArrayTextContent(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":[{"type":"text","text":"line one"},{"type":"text","text":"line two"}]}}]}`)),
			Request:    r,
		}, nil
	})}
	client, err := NewClient(Config{
		Provider:  ProviderModelHub,
		APIURL:    "https://aidp.test/api/modelhub/online/v2/crawl",
		APIKey:    "secret-ak",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	report, err := client.Generate(context.Background(), "analysis input")
	if err != nil {
		t.Fatal(err)
	}
	if report != "line oneline two" {
		t.Fatalf("report = %q", report)
	}
}

func TestAIDPClientMissingContentErrorIncludesSafeShapeOnly(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"id":"chatcmpl_x","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"","reasoning_content":""}}],"usage":{"prompt_tokens":1}}`)),
			Request:    r,
		}, nil
	})}
	client, err := NewClient(Config{
		Provider:  ProviderModelHub,
		APIURL:    "https://aidp.test/api/modelhub/online/v2/crawl",
		APIKey:    "secret-ak",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Generate(context.Background(), "analysis input")
	if err == nil {
		t.Fatal("expected missing content error")
	}
	msg := err.Error()
	for _, want := range []string{"top_level_keys=", "choices_len=1", "message_keys=content,reasoning_content,role", "content_type=string", "content_len=0", "finish_reason=stop"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not include %q", msg, want)
		}
	}
	for _, deny := range []string{"secret-ak", "analysis input", "chatcmpl_x"} {
		if strings.Contains(msg, deny) {
			t.Fatalf("error %q leaks %q", msg, deny)
		}
	}
}

func TestAIDPClientBuildsModelHubRequest(t *testing.T) {
	var gotAK string
	var gotBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAK = r.URL.Query().Get("ak")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":"report text"}}]}`)),
			Request:    r,
		}, nil
	})}

	client, err := NewClient(Config{
		Provider:  ProviderModelHub,
		APIURL:    "https://aidp.test/api/modelhub/online/v2/crawl",
		APIKey:    "ak-test",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	report, err := client.Generate(context.Background(), "analysis input")
	if err != nil {
		t.Fatal(err)
	}
	if report != "report text" {
		t.Fatalf("report = %q", report)
	}
	if gotAK != "ak-test" {
		t.Fatalf("ak = %q", gotAK)
	}
	if gotBody["model"] != "gpt-5.4-2026-03-05" {
		t.Fatalf("model body = %#v", gotBody)
	}
}

func TestAIDPClientRetriesWhenLengthConsumesAllContent(t *testing.T) {
	call := 0
	var seenMaxTokens []int
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		call++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		seenMaxTokens = append(seenMaxTokens, int(body["max_tokens"].(float64)))
		response := `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"","reasoning_content":""}}]}`
		if call == 3 {
			response = `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"report text","reasoning_content":""}}]}`
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(response)),
			Request:    r,
		}, nil
	})}

	client, err := NewClient(Config{
		Provider:  ProviderModelHub,
		APIURL:    "https://aidp.test/api/modelhub/online/v2/crawl",
		APIKey:    "ak-test",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	report, err := client.Generate(context.Background(), "analysis input")
	if err != nil {
		t.Fatal(err)
	}
	if report != "report text" {
		t.Fatalf("report = %q", report)
	}
	if len(seenMaxTokens) != 3 {
		t.Fatalf("calls = %d, want 3", len(seenMaxTokens))
	}
	if seenMaxTokens[0] != 5000 || seenMaxTokens[1] != 10000 || seenMaxTokens[2] != 20000 {
		t.Fatalf("max_tokens sequence = %#v", seenMaxTokens)
	}
}

func TestAIDPClientRetriesOnceWhenValidatorRejectsOutput(t *testing.T) {
	call := 0
	var prompts []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		call++
		var body struct {
			Messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		prompts = append(prompts, body.Messages[0].Content[0].Text)
		response := `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"{\"primary_persona\":\"UNKNOWN\"}","reasoning_content":""}}]}`
		if call == 2 {
			response = `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"{\"primary_persona\":\"PRISM\"}","reasoning_content":""}}]}`
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(response)),
			Request:    r,
		}, nil
	})}

	client, err := NewClient(Config{
		Provider:  ProviderModelHub,
		APIURL:    "https://aidp.test/api/modelhub/online/v2/crawl",
		APIKey:    "ak-test",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 5000,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	report, err := client.GenerateWithValidation(context.Background(), "analysis input", func(content string) error {
		if !strings.Contains(content, `"primary_persona":"PRISM"`) {
			return fmt.Errorf("primary_persona must be one of the BSPI catalog values")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if report != `{"primary_persona":"PRISM"}` {
		t.Fatalf("report = %q", report)
	}
	if len(prompts) != 2 {
		t.Fatalf("calls = %d, want 2", len(prompts))
	}
	if !strings.Contains(prompts[1], "没有通过校验") {
		t.Fatalf("retry prompt = %q", prompts[1])
	}
}

func TestNewClientRejectsUnknownProvider(t *testing.T) {
	_, err := NewClient(Config{
		Provider:  Provider("custom"),
		APIURL:    "https://example.com/api",
		APIKey:    "test-key",
		Model:     "test-model",
		MaxTokens: 5000,
	}, http.DefaultClient)
	if err == nil {
		t.Fatal("expected invalid provider error")
	}
	if err.Error() != `unsupported llm provider "custom"` {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestKimiClientBuildsChatCompletionsRequest(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":"report text"}}]}`)),
			Request:    r,
		}, nil
	})}

	client, err := NewClient(Config{
		Provider:  ProviderKimi,
		APIURL:    "https://api.moonshot.cn/v1/chat/completions",
		APIKey:    "Bearer test-key",
		Model:     "kimi-k2.5",
		MaxTokens: 7000,
	}, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	report, err := client.Generate(context.Background(), "analysis input")
	if err != nil {
		t.Fatal(err)
	}
	if report != "report text" {
		t.Fatalf("report = %q", report)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotBody["model"] != "kimi-k2.5" {
		t.Fatalf("model body = %#v", gotBody)
	}
	if gotBody["max_completion_tokens"] != float64(7000) {
		t.Fatalf("max_completion_tokens = %#v", gotBody["max_completion_tokens"])
	}
	thinking, ok := gotBody["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking body = %#v", gotBody["thinking"])
	}
	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages body = %#v", gotBody["messages"])
	}
	message, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("message body = %#v", messages[0])
	}
	if message["content"] != "analysis input" {
		t.Fatalf("content body = %#v", message["content"])
	}
}
