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
	  "highlight_tags": ["跨团队桥接", "语境切换", "高频协同"],
	  "behavior_vectors": [
	    {
	      "label": "协作方式",
	      "left_pole": "独立成局",
	      "right_pole": "高频协同",
	      "score": 82,
	      "summary": "更容易在多人协同和跨团队桥接场景里发挥影响力。"
	    },
	    {
	      "label": "表达风格",
	      "left_pole": "克制压缩",
	      "right_pole": "高频输出",
	      "score": 71,
	      "summary": "会根据对象切换表达密度，但整体偏主动输出。"
	    },
	    {
	      "label": "决策路径",
	      "left_pole": "证据校准",
	      "right_pole": "直觉快判",
	      "score": 44,
	      "summary": "更倾向先校准语境和事实，再推动形成判断。"
	    },
	    {
	      "label": "推进节奏",
	      "left_pole": "稳态推进",
	      "right_pole": "高压突进",
	      "score": 58,
	      "summary": "整体节奏稳中偏快，遇到窗口期会明显提速。"
	    }
	  ],
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
	if len(got.HighlightTags) != 3 {
		t.Fatalf("highlight_tags = %#v", got.HighlightTags)
	}
	if len(got.BehaviorVectors) != 4 {
		t.Fatalf("behavior_vectors = %#v", got.BehaviorVectors)
	}
	if got.BehaviorVectors[0].Label != "协作方式" || got.BehaviorVectors[0].Score != 82 {
		t.Fatalf("behavior_vector[0] = %#v", got.BehaviorVectors[0])
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
	raw := "```json\n{\n  \"primary_persona\": \"CALM\",\n  \"summary\": \"x\",\n  \"evidence\": [\"x\"],\n  \"communication_style\": \"x\",\n  \"work_preferences\": \"x\",\n  \"blind_spots\": \"x\",\n  \"highlight_tags\": [\"冷静控场\", \"理性沟通\"],\n  \"behavior_vectors\": [\n    {\n      \"label\": \"协作方式\",\n      \"left_pole\": \"独立成局\",\n      \"right_pole\": \"高频协同\",\n      \"score\": 60,\n      \"summary\": \"x\"\n    },\n    {\n      \"label\": \"表达风格\",\n      \"left_pole\": \"克制压缩\",\n      \"right_pole\": \"高频输出\",\n      \"score\": 40,\n      \"summary\": \"x\"\n    },\n    {\n      \"label\": \"决策路径\",\n      \"left_pole\": \"证据校准\",\n      \"right_pole\": \"直觉快判\",\n      \"score\": 35,\n      \"summary\": \"x\"\n    },\n    {\n      \"label\": \"推进节奏\",\n      \"left_pole\": \"稳态推进\",\n      \"right_pole\": \"高压突进\",\n      \"score\": 48,\n      \"summary\": \"x\"\n    }\n  ],\n  \"confidence\": 0.75,\n  \"disclaimer\": \"x\"\n}\n```"
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
	  "highlight_tags": ["冷静控场", "理性沟通"],
	  "behavior_vectors": [
	    {
	      "label": "协作方式",
	      "left_pole": "独立成局",
	      "right_pole": "高频协同",
	      "score": 60,
	      "summary": "x"
	    },
	    {
	      "label": "表达风格",
	      "left_pole": "克制压缩",
	      "right_pole": "高频输出",
	      "score": 40,
	      "summary": "x"
	    },
	    {
	      "label": "决策路径",
	      "left_pole": "证据校准",
	      "right_pole": "直觉快判",
	      "score": 35,
	      "summary": "x"
	    },
	    {
	      "label": "推进节奏",
	      "left_pole": "稳态推进",
	      "right_pole": "高压突进",
	      "score": 48,
	      "summary": "x"
	    }
	  ],
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
	  "highlight_tags": ["冷静控场", "理性沟通"],
	  "behavior_vectors": [
	    {
	      "label": "协作方式",
	      "left_pole": "独立成局",
	      "right_pole": "高频协同",
	      "score": 60,
	      "summary": "x"
	    },
	    {
	      "label": "表达风格",
	      "left_pole": "克制压缩",
	      "right_pole": "高频输出",
	      "score": 40,
	      "summary": "x"
	    },
	    {
	      "label": "决策路径",
	      "left_pole": "证据校准",
	      "right_pole": "直觉快判",
	      "score": 35,
	      "summary": "x"
	    },
	    {
	      "label": "推进节奏",
	      "left_pole": "稳态推进",
	      "right_pole": "高压突进",
	      "score": 48,
	      "summary": "x"
	    }
	  ],
	  "confidence": 1.2,
	  "disclaimer": "x"
	}`
	_, err := ParseLLMResult(raw)
	if err == nil || !strings.Contains(err.Error(), "confidence must be between 0 and 1") {
		t.Fatalf("err = %v", err)
	}
}
