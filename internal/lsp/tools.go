package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/tools"
)

func RegisterLSPTools(hooks *hook.Hooks, m *Manager) {
	hooks.AddFilter(tools.HookTools, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]agent.Tool)
		return append(existing, NewLSPTools(m)...)
	}, 50)
}

func NewLSPTools(m *Manager) []agent.Tool {
	return []agent.Tool{
		newCompletionTool(m),
		newDefinitionTool(m),
		newReferencesTool(m),
		newHoverTool(m),
		newDocumentSymbolTool(m),
		newCodeActionTool(m),
		newDiagnosticsTool(m),
		newFormatTool(m),
		newRenameTool(m),
		newSignatureHelpTool(m),
	}
}

func clientForURI(m *Manager, uri string) (*Client, error) {
	clients, err := m.GetClientForFile(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client for %s: %w", uri, err)
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no LSP server handles %s", uri)
	}
	return clients[0], nil
}

func asString(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func asInt(params map[string]any, key string) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func newCompletionTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_completion",
		"Get auto-completion items at a position in a file. Returns a list of completion suggestions.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
			},
			"required": []string{"uri", "line", "character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			result, err := client.Completion(ctx, uri, pos)
			if err != nil {
				return "", fmt.Errorf("completion failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newDefinitionTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_definition",
		"Go to definition of the symbol at a position in a file. Returns one or more locations.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
			},
			"required": []string{"uri", "line", "character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			result, err := client.Definition(ctx, uri, pos)
			if err != nil {
				return "", fmt.Errorf("definition failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newReferencesTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_references",
		"Find all references to the symbol at a position in a file. Returns a list of locations.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
				"include_declaration": map[string]any{
					"type":        "boolean",
					"description": "Whether to include the declaration itself (default true)",
				},
			},
			"required": []string{"uri", "line", "character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			includeDecl := true
			if v, ok := params["include_declaration"]; ok {
				if b, ok := v.(bool); ok {
					includeDecl = b
				}
			}

			result, err := client.References(ctx, uri, pos, includeDecl)
			if err != nil {
				return "", fmt.Errorf("references failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newHoverTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_hover",
		"Get hover information for the symbol at a position in a file. Returns type info and documentation.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
			},
			"required": []string{"uri", "line", "character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			result, err := client.Hover(ctx, uri, pos)
			if err != nil {
				return "", fmt.Errorf("hover failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newDocumentSymbolTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_document_symbol",
		"Get the document outline (symbols) for a file. Returns a hierarchical list of symbols.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
			},
			"required": []string{"uri"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			result, err := client.DocumentSymbol(ctx, uri)
			if err != nil {
				return "", fmt.Errorf("document symbol failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newCodeActionTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_code_action",
		"Get available code actions (quick fixes, refactorings) for a range in a file.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"start_line": map[string]any{
					"type":        "integer",
					"description": "Zero-based start line of the range",
				},
				"start_character": map[string]any{
					"type":        "integer",
					"description": "Zero-based start character offset",
				},
				"end_line": map[string]any{
					"type":        "integer",
					"description": "Zero-based end line of the range",
				},
				"end_character": map[string]any{
					"type":        "integer",
					"description": "Zero-based end character offset",
				},
				"diagnostics": map[string]any{
					"type":        "array",
					"description": "Diagnostics to consider (optional, uses cached diagnostics if omitted)",
					"items": map[string]any{
						"type": "object",
					},
				},
			},
			"required": []string{"uri", "start_line", "start_character", "end_line", "end_character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			r := Range{
				Start: Position{
					Line:      asInt(params, "start_line"),
					Character: asInt(params, "start_character"),
				},
				End: Position{
					Line:      asInt(params, "end_line"),
					Character: asInt(params, "end_character"),
				},
			}

			var diags []Diagnostic
			if rawDiags, ok := params["diagnostics"]; ok {
				if rawSlice, ok := rawDiags.([]any); ok {
					for _, raw := range rawSlice {
						if rawMap, ok := raw.(map[string]any); ok {
							data, _ := json.Marshal(rawMap)
							var d Diagnostic
							if err := json.Unmarshal(data, &d); err == nil {
								diags = append(diags, d)
							}
						}
					}
				}
			}
			if len(diags) == 0 {
				diags, _ = client.Diagnostics(ctx, uri)
			}

			result, err := client.CodeAction(ctx, uri, r, diags)
			if err != nil {
				return "", fmt.Errorf("code action failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newDiagnosticsTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_diagnostics",
		"Get cached diagnostics (errors, warnings) for a file. Diagnostics are updated asynchronously.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
			},
			"required": []string{"uri"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			result, err := client.Diagnostics(ctx, uri)
			if err != nil {
				return "", fmt.Errorf("diagnostics failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newFormatTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_format",
		"Format a document and return the resulting text edits. Applies default formatting options.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"tab_size": map[string]any{
					"type":        "integer",
					"description": "Number of spaces per tab (default 4)",
				},
				"insert_spaces": map[string]any{
					"type":        "boolean",
					"description": "Use spaces instead of tabs (default true)",
				},
			},
			"required": []string{"uri"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			tabSize := asInt(params, "tab_size")
			if tabSize == 0 {
				tabSize = 4
			}
			insertSpaces := true
			if v, ok := params["insert_spaces"]; ok {
				if b, ok := v.(bool); ok {
					insertSpaces = b
				}
			}

			opts := FormattingOptions{
				TabSize:      tabSize,
				InsertSpaces: insertSpaces,
			}

			result, err := client.Format(ctx, uri, opts)
			if err != nil {
				return "", fmt.Errorf("format failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newRenameTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_rename",
		"Rename a symbol at a position. Returns a workspace edit with all changes needed.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
				"new_name": map[string]any{
					"type":        "string",
					"description": "The new name for the symbol",
				},
			},
			"required": []string{"uri", "line", "character", "new_name"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			newName := asString(params, "new_name")
			if newName == "" {
				return "", fmt.Errorf("new_name is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			result, err := client.Rename(ctx, uri, pos, newName)
			if err != nil {
				return "", fmt.Errorf("rename failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func newSignatureHelpTool(m *Manager) agent.Tool {
	return agent.NewFuncTool(
		"lsp_signature_help",
		"Get function signature help at a position. Returns available overloads and parameter info.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"uri": map[string]any{
					"type":        "string",
					"description": "File URI (e.g. file:///path/to/file.go)",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Zero-based line number",
				},
				"character": map[string]any{
					"type":        "integer",
					"description": "Zero-based character offset on the line",
				},
			},
			"required": []string{"uri", "line", "character"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			uri := asString(params, "uri")
			if uri == "" {
				return "", fmt.Errorf("uri is required")
			}

			client, err := clientForURI(m, uri)
			if err != nil {
				return "", err
			}

			pos := Position{
				Line:      asInt(params, "line"),
				Character: asInt(params, "character"),
			}

			result, err := client.SignatureHelp(ctx, uri, pos)
			if err != nil {
				return "", fmt.Errorf("signature help failed: %w", err)
			}

			return marshalJSON(result)
		},
	)
}

func marshalJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}
