package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogContainsAllPhotoAssets(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	items := All()
	if len(items) != 20 {
		t.Fatalf("catalog size = %d, want 20", len(items))
	}
	for _, item := range items {
		if item.ImageFile == "" {
			t.Fatalf("%s missing image file", item.Shorthand)
		}
		if _, err := os.Stat(filepath.Join(root, "photos", item.ImageFile)); err != nil {
			t.Fatalf("%s missing photo %s: %v", item.Shorthand, item.ImageFile, err)
		}
	}
}

func TestParseLLMResultBuildsStructuredPersonaResult(t *testing.T) {
	raw := `{
	  "primary_persona": "PRISM",
	  "summary": "跨文化适应能力强，协作语境切换自然。",
	  "evidence": ["经常在不同对象之间切换表达方式", "跨团队协作密集"],
	  "communication_style": "会主动翻译语境差异，让不同角色更快对齐。",
	  "work_preferences": "偏好多方协同、跨语境问题和需要桥接认知差异的场景。",
	  "blind_spots": "容易长期适配别人，偶尔忽略自己的稳定表达方式。",
	  "confidence": 0.86,
	  "disclaimer": "仅基于授权数据的行为风格观察，不是心理诊断。"
	}`

	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryPersona.Shorthand != "PRISM" {
		t.Fatalf("shorthand = %q", got.PrimaryPersona.Shorthand)
	}
	if got.PrimaryPersona.ImageURL != "/assets/photos/PRISM.png" {
		t.Fatalf("image_url = %q", got.PrimaryPersona.ImageURL)
	}
	if !strings.Contains(got.PrimaryPersona.CanonicalDescription, "跨文化") {
		t.Fatalf("canonical_description = %q", got.PrimaryPersona.CanonicalDescription)
	}
	if len(got.Analysis.Evidence) != 2 {
		t.Fatalf("evidence = %#v", got.Analysis.Evidence)
	}
	if got.Analysis.Confidence != 0.86 {
		t.Fatalf("confidence = %v", got.Analysis.Confidence)
	}
}

func TestParseLLMResultRejectsUnknownPersona(t *testing.T) {
	raw := `{
	  "primary_persona": "UNKNOWN",
	  "summary": "x",
	  "evidence": ["x"],
	  "communication_style": "x",
	  "work_preferences": "x",
	  "blind_spots": "x",
	  "confidence": 0.42,
	  "disclaimer": "x"
	}`

	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "not in the BSPI catalog") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseLLMResultAcceptsJSONCodeFence(t *testing.T) {
	raw := "```json\n{\n  \"primary_persona\": \"CALM\",\n  \"summary\": \"x\",\n  \"evidence\": [\"x\"],\n  \"communication_style\": \"x\",\n  \"work_preferences\": \"x\",\n  \"blind_spots\": \"x\",\n  \"confidence\": 0.75,\n  \"disclaimer\": \"x\"\n}\n```"
	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryPersona.Shorthand != "CALM" {
		t.Fatalf("shorthand = %q", got.PrimaryPersona.Shorthand)
	}
}

func TestParseLLMResultAcceptsStringConfidence(t *testing.T) {
	raw := `{
	  "primary_persona": "CALM",
	  "summary": "x",
	  "evidence": ["x"],
	  "communication_style": "x",
	  "work_preferences": "x",
	  "blind_spots": "x",
	  "confidence": "0.64",
	  "disclaimer": "x"
	}`
	got, err := ParseLLMResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Analysis.Confidence != 0.64 {
		t.Fatalf("confidence = %v", got.Analysis.Confidence)
	}
}

func TestParseLLMResultRejectsOutOfRangeConfidence(t *testing.T) {
	raw := `{
	  "primary_persona": "CALM",
	  "summary": "x",
	  "evidence": ["x"],
	  "communication_style": "x",
	  "work_preferences": "x",
	  "blind_spots": "x",
	  "confidence": 1.2,
	  "disclaimer": "x"
	}`
	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "confidence must be between 0 and 1") {
		t.Fatalf("err = %v", err)
	}
}
