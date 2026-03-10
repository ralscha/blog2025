package codemode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/preblog/codemode/internal/catalog"
	"github.com/preblog/codemode/internal/sandbox"
)

type SearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type ExecuteInput struct {
	Code string `json:"code"`
}

type Toolset struct {
	Catalog []catalog.ToolInfo
	Sandbox *sandbox.Sandbox
	Tracef  func(string, ...any)
}

func (t *Toolset) SearchDefinition() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
		Name:        "search",
		Description: anthropic.String("Call this to discover relevant helpers for the user's task. The result includes the JavaScript helper definitions you should use in execute(code)."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"query": map[string]any{"type": "string", "description": "A concise natural-language summary of the user's task or the helpers you need to find."},
				"limit": map[string]any{"type": "integer", "description": "Maximum results to return."},
			},
			Required: []string{"query"},
		},
	}}
}

func (t *Toolset) ExecuteDefinition() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
		Name:        "execute",
		Description: anthropic.String("Execute synchronous JavaScript only. The sandbox supports ECMAScript 14 (ES2023) only, and the flow is sync so it does not support async or await. The sandbox does not support any Web APIs. Use the helper definitions returned by search, for example demo_add_numbers({...}). Keep the code minimal and do not write comments."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"code": map[string]any{"type": "string", "description": "JavaScript body to execute inside a synchronous IIFE. End by returning a value. Do not write comments."},
			},
			Required: []string{"code"},
		},
	}}
}

func (t *Toolset) Definitions() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{t.SearchDefinition(), t.ExecuteDefinition()}
}

func (t *Toolset) Description() string {
	return catalog.HelperDefinitions(t.Catalog)
}

func (t *Toolset) Execute(ctx context.Context, name string, input json.RawMessage) (string, bool, error) {
	switch name {
	case "search":
		var args SearchInput
		if err := json.Unmarshal(input, &args); err != nil {
			return "", true, fmt.Errorf("parse search input: %w", err)
		}
		matches := catalog.Search(t.Catalog, args.Query, args.Limit)
		payload, err := json.Marshal(map[string]any{
			"api_definition": catalog.HelperDefinitions(matches),
		})
		if err != nil {
			return "", true, err
		}
		if t.Tracef != nil {
			t.Tracef("tool search query=%q results=%d\n%s", args.Query, len(matches), indent(string(payload), "  "))
		}
		return string(payload), false, nil
	case "execute":
		var args ExecuteInput
		if err := json.Unmarshal(input, &args); err != nil {
			return "", true, fmt.Errorf("parse execute input: %w", err)
		}
		result, err := t.Sandbox.Execute(ctx, args.Code)
		if err != nil {
			if t.Tracef != nil {
				t.Tracef("tool execute error=%q", err.Error())
			}
			return "", true, err
		}
		payload, err := json.Marshal(result)
		if err != nil {
			return "", true, err
		}
		if t.Tracef != nil {
			t.Tracef("tool execute ok logs=%d\n%s", len(result.Logs), indent(string(payload), "  "))
		}
		return string(payload), false, nil
	default:
		return "", true, fmt.Errorf("unknown tool %q", name)
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
