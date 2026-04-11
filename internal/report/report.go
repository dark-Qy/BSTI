package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"feishu-personality-agent/internal/persona"
)

type Paths struct {
	Markdown string
	HTML     string
}

func WriteLocal(dir string, result persona.Result) (Paths, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return Paths{}, err
	}
	paths := Paths{
		Markdown: filepath.Join(dir, "report.md"),
		HTML:     filepath.Join(dir, "report.html"),
	}
	markdown := renderMarkdown(result)
	if err := os.WriteFile(paths.Markdown, []byte(markdown), 0600); err != nil {
		return Paths{}, err
	}
	page := renderHTML(result)
	if err := os.WriteFile(paths.HTML, []byte(page), 0600); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func renderMarkdown(result persona.Result) string {
	var b strings.Builder
	b.WriteString("# BSPI Personality Report\n\n")
	b.WriteString("## Primary Persona\n")
	b.WriteString("- Shorthand: " + result.PrimaryPersona.Shorthand + "\n")
	b.WriteString("- 中文名: " + result.PrimaryPersona.ChineseLabel + "\n")
	b.WriteString("- 字节范儿维度: " + result.PrimaryPersona.ByteStyleDimension + "\n")
	b.WriteString("- 分析维度: " + result.PrimaryPersona.AnalysisDimension + "\n")
	b.WriteString("- 一句话画像: " + result.PrimaryPersona.OneLiner + "\n")
	b.WriteString("- 图片: " + result.PrimaryPersona.ImageURL + "\n\n")
	b.WriteString("## 官方人格定义\n")
	b.WriteString(result.PrimaryPersona.CanonicalDescription + "\n\n")
	b.WriteString("## 个体分析摘要\n")
	b.WriteString(result.Analysis.Summary + "\n\n")
	b.WriteString("## 证据\n")
	for _, item := range result.Analysis.Evidence {
		b.WriteString("- " + item + "\n")
	}
	b.WriteString("\n## 沟通风格\n")
	b.WriteString(result.Analysis.CommunicationStyle + "\n\n")
	b.WriteString("## 工作偏好\n")
	b.WriteString(result.Analysis.WorkPreferences + "\n\n")
	b.WriteString("## 风险与盲区\n")
	b.WriteString(result.Analysis.BlindSpots + "\n\n")
	b.WriteString("## 置信度\n")
	b.WriteString(fmt.Sprintf("%.2f", result.Analysis.Confidence) + "\n\n")
	b.WriteString("## 免责声明\n")
	b.WriteString(result.Analysis.Disclaimer + "\n")
	return b.String()
}

func renderHTML(result persona.Result) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>BSPI Personality Report</title><style>")
	b.WriteString("body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;max-width:960px;margin:32px auto;line-height:1.6;padding:0 20px;color:#111}")
	b.WriteString(".hero{display:grid;grid-template-columns:minmax(240px,320px) 1fr;gap:24px;align-items:start;margin-bottom:28px}")
	b.WriteString(".hero img{width:100%;height:auto;border-radius:8px;display:block}")
	b.WriteString(".eyebrow{font-size:12px;text-transform:uppercase;color:#666;margin-bottom:8px}")
	b.WriteString("h1,h2{margin:0 0 12px} h2{margin-top:28px}")
	b.WriteString(".meta{margin:0 0 16px;padding:0;list-style:none}.meta li{margin:4px 0}")
	b.WriteString(".summary,.section{margin-bottom:20px}.section p{margin:0}.evidence{padding-left:18px}")
	b.WriteString("</style></head><body>")
	b.WriteString("<div class=\"hero\">")
	b.WriteString("<div><img src=\"" + html.EscapeString(result.PrimaryPersona.ImageURL) + "\" alt=\"" + html.EscapeString(result.PrimaryPersona.Shorthand) + "\"></div>")
	b.WriteString("<div>")
	b.WriteString("<div class=\"eyebrow\">BSPI Top1 Persona</div>")
	b.WriteString("<h1>" + html.EscapeString(result.PrimaryPersona.Shorthand) + " / " + html.EscapeString(result.PrimaryPersona.ChineseLabel) + "</h1>")
	b.WriteString("<p class=\"summary\">" + html.EscapeString(result.PrimaryPersona.OneLiner) + "</p>")
	b.WriteString("<ul class=\"meta\">")
	b.WriteString("<li><strong>字节范儿维度：</strong>" + html.EscapeString(result.PrimaryPersona.ByteStyleDimension) + "</li>")
	b.WriteString("<li><strong>分析维度：</strong>" + html.EscapeString(result.PrimaryPersona.AnalysisDimension) + "</li>")
	b.WriteString("<li><strong>置信度：</strong>" + fmt.Sprintf("%.2f", result.Analysis.Confidence) + "</li>")
	b.WriteString("</ul>")
	b.WriteString("</div></div>")
	b.WriteString("<div class=\"section\"><h2>官方人格定义</h2><p>" + html.EscapeString(result.PrimaryPersona.CanonicalDescription) + "</p></div>")
	b.WriteString("<div class=\"section\"><h2>个体分析摘要</h2><p>" + html.EscapeString(result.Analysis.Summary) + "</p></div>")
	b.WriteString("<div class=\"section\"><h2>证据</h2><ul class=\"evidence\">")
	for _, item := range result.Analysis.Evidence {
		b.WriteString("<li>" + html.EscapeString(item) + "</li>")
	}
	b.WriteString("</ul></div>")
	b.WriteString("<div class=\"section\"><h2>沟通风格</h2><p>" + html.EscapeString(result.Analysis.CommunicationStyle) + "</p></div>")
	b.WriteString("<div class=\"section\"><h2>工作偏好</h2><p>" + html.EscapeString(result.Analysis.WorkPreferences) + "</p></div>")
	b.WriteString("<div class=\"section\"><h2>风险与盲区</h2><p>" + html.EscapeString(result.Analysis.BlindSpots) + "</p></div>")
	b.WriteString("<div class=\"section\"><h2>免责声明</h2><p>" + html.EscapeString(result.Analysis.Disclaimer) + "</p></div>")
	b.WriteString("</body></html>")
	return b.String()
}
