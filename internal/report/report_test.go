package report

import (
	"os"
	"strings"
	"testing"
)

func TestWriteLocalReportWritesMarkdownAndHTML(t *testing.T) {
	dir := t.TempDir()
	paths, err := WriteLocal(dir, "# Title\n\nbody")
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(paths.HTML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(htmlBytes), "<h1>Title</h1>") {
		t.Fatalf("html = %s", string(htmlBytes))
	}
	if _, err := os.Stat(paths.Markdown); err != nil {
		t.Fatal(err)
	}
}
