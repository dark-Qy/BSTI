package report

import (
	"os"
	"strings"
	"testing"

	"feishu-personality-agent/internal/persona"
)

func TestWriteLocalReportWritesStructuredMarkdownAndHTML(t *testing.T) {
	dir := t.TempDir()
	result := persona.Result{
		PrimaryPersona: persona.Primary{
			Shorthand:            "PRISM",
			ChineseLabel:         "变色龙",
			ImageURL:             "/assets/photos/PRISM.png",
			ByteStyleDimension:   "多元兼容",
			AnalysisDimension:    "多元文化适应力",
			OneLiner:             "多线程文化模拟器，每个频道都是真的",
			CanonicalDescription: "PRISM 擅长跨文化切换与桥接。",
		},
		Analysis: persona.Analysis{
			Summary:            "跨文化语境切换自然，适合做协作桥梁。",
			Evidence:           []string{"频繁在不同协作对象之间切换表达方式", "跨团队沟通密度高"},
			CommunicationStyle: "先理解对方语境，再翻译回共同问题。",
			WorkPreferences:    "偏好多方协作与需要桥接认知差异的任务。",
			BlindSpots:         "可能长期适配别人而忽略自己的固定表达方式。",
			Confidence:         "高",
			Disclaimer:         "仅基于授权数据的行为风格观察。",
		},
	}

	paths, err := WriteLocal(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(paths.HTML)
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	for _, want := range []string{"PRISM / 变色龙", "/assets/photos/PRISM.png", "官方人格定义", "个体分析摘要", "跨文化切换与桥接"} {
		if !strings.Contains(html, want) {
			t.Fatalf("html missing %q: %s", want, html)
		}
	}
	mdBytes, err := os.ReadFile(paths.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	md := string(mdBytes)
	if !strings.Contains(md, "## Primary Persona") || !strings.Contains(md, "PRISM") {
		t.Fatalf("markdown = %s", md)
	}
}
