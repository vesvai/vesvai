package web

import (
	"strconv"
	"testing"
)

func TestParseDuckDuckGoResults(t *testing.T) {
	html := `<html>
<body>
<div class="result">
  <a class="result__a" href="https://example.com">Example Site</a>
  <a class="result__snippet" href="https://example.com">This is an example snippet</a>
</div>
<div class="result">
  <a class="result__a" href="https://golang.org">The Go Language</a>
  <div class="result__snippet">Go is a programming language</div>
</div>
</body>
</html>`

	results := parseDuckDuckGoResults(html, 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Example Site" {
		t.Errorf("expected 'Example Site', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("expected 'https://example.com', got %q", results[0].URL)
	}
	if results[0].Snippet != "This is an example snippet" {
		t.Errorf("expected 'This is an example snippet', got %q", results[0].Snippet)
	}

	if results[1].Title != "The Go Language" {
		t.Errorf("expected 'The Go Language', got %q", results[1].Title)
	}
	if results[1].URL != "https://golang.org" {
		t.Errorf("expected 'https://golang.org', got %q", results[1].URL)
	}
	if results[1].Snippet != "Go is a programming language" {
		t.Errorf("expected 'Go is a programming language', got %q", results[1].Snippet)
	}
}

func TestParseDuckDuckGoResultsEmpty(t *testing.T) {
	results := parseDuckDuckGoResults("<html><body>no results</body></html>", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseDuckDuckGoResultsMaxResults(t *testing.T) {
	html := "<html><body>"
	for i := 0; i < 5; i++ {
		html += `<div class="result"><a class="result__a" href="https://example.com/` + strconv.Itoa(i) + `">Result ` + strconv.Itoa(i) + `</a></div>`
	}
	html += "</body></html>"

	results := parseDuckDuckGoResults(html, 3)
	if len(results) != 3 {
		t.Errorf("expected 3 results (max), got %d", len(results))
	}
}

func TestParseDuckDuckGoResultsMalformed(t *testing.T) {
	results := parseDuckDuckGoResults("not valid html", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for malformed html, got %d", len(results))
	}
}

func TestCleanDuckDuckGoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com", "https://example.com"},
		{"//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage", "https://example.com/page"},
		{"https://duckduckgo.com/l/?uddg=https%3A%2F%2Fgolang.org", "https://golang.org"},
		{"javascript:;", "javascript:;"},
	}
	for _, tt := range tests {
		got := cleanDuckDuckGoURL(tt.input)
		if got != tt.want {
			t.Errorf("cleanDuckDuckGoURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSearchToolMissingQuery(t *testing.T) {
	tool := searchTool(nil)
	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}
