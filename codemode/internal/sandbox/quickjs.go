package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/preblog/codemode/internal/catalog"
	"github.com/preblog/codemode/internal/mcpdemo"
	"modernc.org/quickjs"
)

type Result struct {
	Logs  []string `json:"logs"`
	Value any      `json:"value"`
}

type Sandbox struct {
	catalog         []catalog.ToolInfo
	runtime         *mcpdemo.Runtime
	evalTimeout     time.Duration
	memoryLimitByte uintptr
}

func New(items []catalog.ToolInfo, runtime *mcpdemo.Runtime, evalTimeout time.Duration, memoryLimit uintptr) *Sandbox {
	return &Sandbox{catalog: items, runtime: runtime, evalTimeout: evalTimeout, memoryLimitByte: memoryLimit}
}

func (s *Sandbox) Execute(ctx context.Context, code string) (result Result, err error) {
	vm, err := quickjs.NewVM()
	if err != nil {
		return Result{}, fmt.Errorf("create quickjs vm: %w", err)
	}
	defer func() {
		if closeErr := vm.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close quickjs vm: %w", closeErr)
		}
	}()

	vm.SetMemoryLimit(s.memoryLimitByte)
	if err := vm.SetEvalTimeout(s.evalTimeout); err != nil {
		return Result{}, fmt.Errorf("set eval timeout: %w", err)
	}

	logs := []string{}
	if err := vm.RegisterFunc("__host_log", func(payload string) {
		logs = append(logs, payload)
	}, false); err != nil {
		return Result{}, fmt.Errorf("register logger: %w", err)
	}

	for _, item := range s.catalog {
		toolName := item.Name
		bridgeName := "__bridge_" + item.Callable
		if err := vm.RegisterFunc(bridgeName, func(payload string) string {
			var args map[string]any
			if err := json.Unmarshal([]byte(payload), &args); err != nil {
				return marshalBridgeResponse(nil, fmt.Sprintf("parse tool args: %v", err))
			}
			toolCtx, cancel := context.WithTimeout(ctx, s.evalTimeout)
			defer cancel()
			result, err := s.runtime.CallTool(toolCtx, toolName, args)
			if err != nil {
				return marshalBridgeResponse(nil, err.Error())
			}
			if result.IsError {
				return marshalBridgeResponse(nil, flattenContent(result))
			}
			if result.StructuredContent != nil {
				return marshalBridgeResponse(result.StructuredContent, "")
			}
			return marshalBridgeResponse(map[string]any{"content": flattenContent(result)}, "")
		}, false); err != nil {
			return Result{}, fmt.Errorf("register %s: %w", bridgeName, err)
		}
	}

	prelude := s.prelude()
	wrapped := prelude + "\n(() => {\n" + code + "\n})()"
	value, err := vm.Eval(wrapped, quickjs.EvalGlobal)
	if err != nil {
		return Result{}, fmt.Errorf("execute javascript: %w", err)
	}

	return Result{
		Logs:  logs,
		Value: value,
	}, nil
}

func (s *Sandbox) prelude() string {
	var builder strings.Builder
	builder.WriteString("const console = {\n")
	builder.WriteString("  log: (...args) => __host_log(JSON.stringify(args)),\n")
	builder.WriteString("};\n")
	for _, item := range s.catalog {
		fmt.Fprintf(&builder, "function %s(args) { const response = JSON.parse(__bridge_%s(JSON.stringify(args || {}))); if (response.error) { throw new Error(response.error); } return response.value; }\n", item.Callable, item.Callable)
	}
	return builder.String()
}

func flattenContent(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		switch content := content.(type) {
		case *mcp.TextContent:
			parts = append(parts, content.Text)
		default:
			b, _ := json.Marshal(content)
			parts = append(parts, string(b))
		}
	}
	return strings.Join(parts, "\n")
}

func marshalBridgeResponse(value any, errText string) string {
	payload := map[string]any{"value": value, "error": errText}
	b, err := json.Marshal(payload)
	if err != nil {
		fallback, _ := json.Marshal(map[string]any{"value": nil, "error": fmt.Sprintf("marshal bridge response: %v", err)})
		return string(fallback)
	}
	return string(b)
}
