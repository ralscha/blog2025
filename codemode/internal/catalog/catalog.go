package catalog

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolInfo struct {
	Server       string         `json:"server"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Callable     string         `json:"callable"`
	InputSchema  map[string]any `json:"input_schema,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
}

type Lister interface {
	ListTools(context.Context) ([]*mcp.Tool, error)
}

func Load(ctx context.Context, serverName string, lister Lister) ([]ToolInfo, error) {
	tools, err := lister.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ToolInfo, 0, len(tools))
	for _, tool := range tools {
		items = append(items, ToolInfo{
			Server:       serverName,
			Name:         tool.Name,
			Description:  tool.Description,
			Callable:     fmt.Sprintf("%s_%s", serverName, tool.Name),
			InputSchema:  schemaToMap(tool.InputSchema),
			OutputSchema: schemaToMap(tool.OutputSchema),
		})
	}
	slices.SortFunc(items, func(a, b ToolInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return items, nil
}

func Search(items []ToolInfo, query string, limit int) []ToolInfo {
	if limit <= 0 {
		limit = 8
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		if len(items) <= limit {
			return append([]ToolInfo(nil), items...)
		}
		return append([]ToolInfo(nil), items[:limit]...)
	}

	terms := expandSearchTerms(searchTerms(query))
	type scoredMatch struct {
		item  ToolInfo
		score int
	}
	matches := make([]scoredMatch, 0, len(items))
	for _, item := range items {
		score := searchScore(item, terms)
		if score > 0 {
			matches = append(matches, scoredMatch{item: item, score: score})
		}
	}
	slices.SortFunc(matches, func(a, b scoredMatch) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score),
			cmp.Compare(a.item.Name, b.item.Name),
			cmp.Compare(a.item.Callable, b.item.Callable),
		)
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	results := make([]ToolInfo, 0, len(matches))
	for _, match := range matches {
		results = append(results, match.item)
	}
	return results
}

func HelperDefinitions(items []ToolInfo) string {
	var builder strings.Builder
	if len(items) == 0 {
		builder.WriteString("// No helper functions matched this search.\n")
		return builder.String()
	}

	for _, item := range items {
		builder.WriteString("/**\n")
		if desc := sanitizeJSDocText(item.Description); desc != "" {
			builder.WriteString(" * ")
			builder.WriteString(desc)
			builder.WriteString("\n")
		}
		builder.WriteString(" * @param ")
		builder.WriteString(jsDocObjectType(item.InputSchema))
		builder.WriteString(" args\n")
		writeSchemaFieldDescriptions(&builder, "Input fields", "args", item.InputSchema)
		builder.WriteString(" * @returns {")
		builder.WriteString(jsDocType(item.OutputSchema))
		builder.WriteString("}\n")
		writeSchemaFieldDescriptions(&builder, "Output fields", "result", item.OutputSchema)
		builder.WriteString(" */\n")
		builder.WriteString("function ")
		builder.WriteString(item.Callable)
		builder.WriteString("(args) {}\n\n")
	}

	return builder.String()
}

func writeSchemaFieldDescriptions(builder *strings.Builder, heading string, root string, schema map[string]any) {
	lines := schemaFieldDescriptions(root, schema)
	if len(lines) == 0 {
		return
	}
	builder.WriteString(" * ")
	builder.WriteString(heading)
	builder.WriteString(":\n")
	for _, line := range lines {
		builder.WriteString(" *   - ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
}

func schemaFieldDescriptions(root string, schema map[string]any) []string {
	if len(schema) == 0 {
		return nil
	}
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return nil
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	slices.Sort(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		propertySchema, _ := properties[name].(map[string]any)
		path := jsPropertyPath(root, name)
		if desc := sanitizeJSDocText(schemaDescription(propertySchema)); desc != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", path, desc))
		}
		lines = append(lines, schemaFieldDescriptionsForProperty(path, propertySchema)...)
	}
	return lines
}

func schemaFieldDescriptionsForProperty(path string, schema map[string]any) []string {
	if len(schema) == 0 {
		return nil
	}

	typeName, _ := schema["type"].(string)
	switch typeName {
	case "object":
		return schemaFieldDescriptions(path, schema)
	case "array":
		itemSchema, _ := schema["items"].(map[string]any)
		itemPath := path + "[]"
		lines := make([]string, 0, 1)
		if desc := sanitizeJSDocText(schemaDescription(itemSchema)); desc != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", itemPath, desc))
		}
		return append(lines, schemaFieldDescriptionsForProperty(itemPath, itemSchema)...)
	default:
		return nil
	}
}

func schemaDescription(schema map[string]any) string {
	if len(schema) == 0 {
		return ""
	}
	description, _ := schema["description"].(string)
	return description
}

func searchTerms(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	terms := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		terms = append(terms, field)
	}
	return terms
}

func expandSearchTerms(terms []string) []string {
	if len(terms) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(terms)+8)
	expanded := make([]string, 0, len(terms)+8)
	for _, term := range terms {
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		expanded = append(expanded, term)
	}

	if containsAnyTerm(seen, "shipping", "shipment", "parcel", "carrier", "carriers", "rate", "rates", "quote", "quotes") {
		for _, related := range []string{"carrier", "quote", "delivery", "surcharge", "shipping"} {
			if _, ok := seen[related]; ok {
				continue
			}
			seen[related] = struct{}{}
			expanded = append(expanded, related)
		}
	}

	return expanded
}

func containsAnyTerm(terms map[string]struct{}, candidates ...string) bool {
	for _, candidate := range candidates {
		if _, ok := terms[candidate]; ok {
			return true
		}
	}
	return false
}

func searchScore(item ToolInfo, terms []string) int {
	if len(terms) == 0 {
		return 0
	}
	name := strings.ToLower(item.Name)
	callable := strings.ToLower(item.Callable)
	description := strings.ToLower(item.Description)
	schema := strings.ToLower(compactSchema(item.InputSchema))
	server := strings.ToLower(item.Server)

	score := 0
	for _, term := range terms {
		switch {
		case strings.Contains(callable, term):
			score += 6
		case strings.Contains(name, term):
			score += 5
		case strings.Contains(description, term):
			score += 3
		case strings.Contains(server, term):
			score += 2
		case strings.Contains(schema, strconv.Quote(term)) || strings.Contains(schema, term):
			score += 1
		}
	}
	return score
}

func schemaToMap(input any) map[string]any {
	if input == nil {
		return nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return map[string]any{"raw": fmt.Sprintf("%v", input)}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"raw": string(b)}
	}
	return out
}

func compactSchema(schema map[string]any) string {
	if len(schema) == 0 {
		return ""
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	return string(b)
}

func jsDocObjectType(schema map[string]any) string {
	if len(schema) == 0 {
		return "{Record<string, any>}"
	}
	return "{" + jsDocType(schema) + "}"
}

func jsDocObjectLiteral(schema map[string]any) string {
	if len(schema) == 0 {
		return "Record<string, any>"
	}
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return "Record<string, any>"
	}

	required := map[string]struct{}{}
	if values, ok := schema["required"].([]any); ok {
		for _, value := range values {
			if name, ok := value.(string); ok {
				required[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	slices.Sort(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		propertySchema, _ := properties[name].(map[string]any)
		optional := ""
		if _, ok := required[name]; !ok {
			optional = "?"
		}
		parts = append(parts, fmt.Sprintf("%s%s: %s", jsPropertyName(name), optional, jsDocType(propertySchema)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func jsDocType(schema map[string]any) string {
	if len(schema) == 0 {
		return "any"
	}
	if enumValues, ok := schema["enum"].([]any); ok && len(enumValues) > 0 {
		literals := make([]string, 0, len(enumValues))
		for _, value := range enumValues {
			switch value := value.(type) {
			case string:
				literals = append(literals, fmt.Sprintf("%q", value))
			case float64:
				literals = append(literals, strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%f", value), "0"), "."))
			case bool:
				literals = append(literals, fmt.Sprintf("%t", value))
			default:
				literals = append(literals, "any")
			}
		}
		return strings.Join(literals, " | ")
	}

	if unionTypes, ok := schema["type"].([]any); ok && len(unionTypes) > 0 {
		parts := make([]string, 0, len(unionTypes))
		for _, value := range unionTypes {
			if name, ok := value.(string); ok {
				parts = append(parts, jsDocType(map[string]any{"type": name}))
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " | ")
		}
	}

	typeName, _ := schema["type"].(string)
	switch typeName {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	case "array":
		itemSchema, _ := schema["items"].(map[string]any)
		return "Array<" + jsDocType(itemSchema) + ">"
	case "object":
		return jsDocObjectLiteral(schema)
	default:
		return "any"
	}
}

func jsPropertyName(name string) string {
	if isValidJSIdentifier(name) {
		return name
	}
	return fmt.Sprintf("%q", name)
}

func jsPropertyPath(root string, name string) string {
	if root == "" {
		return jsPropertyName(name)
	}
	if isValidJSIdentifier(name) {
		return root + "." + name
	}
	return fmt.Sprintf("%s[%q]", root, name)
}

func isValidJSIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && r != '$' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && r != '$' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func sanitizeJSDocText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	text = strings.ReplaceAll(text, "*/", "* /")
	return text
}
