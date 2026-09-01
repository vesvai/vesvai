package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func fetchTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"web-fetch",
		"Fetch a URL and return its content. By default, HTML pages are converted to clean Markdown for easier reading. Set 'raw' to true to return the raw HTML response without conversion. For plain text URLs, the raw text is returned either way. The response includes the URL, status code, and content type. Use this tool to read documentation, API responses, or any web page content. The request has a 30-second timeout. BINARY content (images, PDFs, etc.) will be noted but not returned.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch. Must include the protocol (http:// or https://). Example: 'https://pkg.go.dev/fmt', 'https://example.com/api/docs'.",
				},
				"raw": map[string]any{
					"type":        "boolean",
					"description": "If true, return the raw HTML response without converting to Markdown. Default is false. Useful when you need to inspect the exact HTML structure or when the Markdown conversion loses important information.",
				},
			},
			"required": []string{"url"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				URL string `json:"url"`
				Raw bool   `json:"raw"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("web-fetch: invalid arguments: %w", err)
			}
			if params.URL == "" {
				return "", fmt.Errorf("web-fetch: url is required")
			}

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
			if err != nil {
				return "", fmt.Errorf("web-fetch: create request: %w", err)
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Vesvai/1.0)")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("web-fetch: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
			if err != nil {
				return "", fmt.Errorf("web-fetch: read body: %w", err)
			}

			contentType := resp.Header.Get("Content-Type")
			if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
				contentType = strings.TrimSpace(contentType[:idx])
			}

			out := fmt.Sprintf("URL: %s\nStatus: %d\nContent-Type: %s\nSize: %d bytes\n\n",
				params.URL, resp.StatusCode, contentType, len(body))

			if !strings.HasPrefix(contentType, "text/") && !strings.Contains(contentType, "html") {
				out += "(binary content, not displayed)\n"
				return out, nil
			}

			if strings.Contains(contentType, "html") || isHTML(body) {
				if params.Raw {
					out += string(body)
				} else {
					markdown, err := md.ConvertString(string(body))
					if err != nil {
						return "", fmt.Errorf("web-fetch: convert to markdown: %w", err)
					}
					out += markdown
				}
			} else {
				out += string(body)
			}

			return out, nil
		},
	)
}

func isHTML(data []byte) bool {
	check := string(data)
	if len(check) > 1024 {
		check = check[:1024]
	}
	lower := strings.ToLower(check)
	return strings.Contains(lower, "<!doctype html") ||
		strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<head") ||
		strings.Contains(lower, "<body")
}
