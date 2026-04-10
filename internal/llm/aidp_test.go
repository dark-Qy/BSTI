package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
