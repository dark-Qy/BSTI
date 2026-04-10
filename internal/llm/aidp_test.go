package llm

import (
	"bytes"
	"context"
	"encoding/json"
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
	client := NewAIDPClient(Config{
		URL:       "https://aidp.test/api/modelhub/online/v2/crawl",
		AK:        "secret-ak",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)

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
	client := NewAIDPClient(Config{
		URL:       "https://aidp.test/api/modelhub/online/v2/crawl",
		AK:        "secret-ak",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)

	_, err := client.Generate(context.Background(), "analysis input")
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

	client := NewAIDPClient(Config{
		URL:       "https://aidp.test/api/modelhub/online/v2/crawl",
		AK:        "ak-test",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
		Stream:    false,
	}, httpClient)

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

	client := NewAIDPClient(Config{
		URL:       "https://aidp.test/api/modelhub/online/v2/crawl",
		AK:        "ak-test",
		Model:     "gpt-5.4-2026-03-05",
		MaxTokens: 500,
	}, httpClient)

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
