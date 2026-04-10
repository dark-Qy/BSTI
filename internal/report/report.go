package report

import (
	"html"
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	Markdown string
	HTML     string
}

func WriteLocal(dir, markdown string) (Paths, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return Paths{}, err
	}
	paths := Paths{
		Markdown: filepath.Join(dir, "report.md"),
		HTML:     filepath.Join(dir, "report.html"),
	}
	if err := os.WriteFile(paths.Markdown, []byte(markdown), 0600); err != nil {
		return Paths{}, err
	}
	page := "<!doctype html><html><head><meta charset=\"utf-8\"><title>Feishu Personality Report</title><style>body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;max-width:880px;margin:40px auto;line-height:1.6;padding:0 20px}pre{white-space:pre-wrap}</style></head><body>" + markdownToHTML(markdown) + "</body></html>"
	if err := os.WriteFile(paths.HTML, []byte(page), 0600); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func markdownToHTML(markdown string) string {
	var out strings.Builder
	paragraph := []string{}
	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out.WriteString("<p>")
		out.WriteString(html.EscapeString(strings.Join(paragraph, " ")))
		out.WriteString("</p>\n")
		paragraph = paragraph[:0]
	}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flushParagraph()
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			flushParagraph()
			out.WriteString("<h1>")
			out.WriteString(html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))))
			out.WriteString("</h1>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			flushParagraph()
			out.WriteString("<h2>")
			out.WriteString(html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))))
			out.WriteString("</h2>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			flushParagraph()
			out.WriteString("<p>• ")
			out.WriteString(html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
			out.WriteString("</p>\n")
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	flushParagraph()
	return out.String()
}
