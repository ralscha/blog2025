package anthropicloop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/preblog/codemode/internal/codemode"
)

type Result struct {
	Text  string
	Trace []string
}

type Runner struct {
	client    anthropic.Client
	model     string
	maxTurns  int
	tools     *codemode.Toolset
	verbose   bool
	debugHTTP bool
	trace     []string
}

func (r *Runner) snapshotResult(text string) Result {
	return Result{Text: text, Trace: append([]string(nil), r.trace...)}
}

func New(apiKey, model string, maxTurns int, verbose bool, debugHTTP bool, captureHTTPInTrace bool, tools *codemode.Toolset) *Runner {
	r := &Runner{
		model:     model,
		maxTurns:  maxTurns,
		tools:     tools,
		verbose:   verbose,
		debugHTTP: debugHTTP,
	}
	options := []option.RequestOption{option.WithAPIKey(apiKey)}
	if debugHTTP {
		logger := log.New(os.Stderr, "[anthropic-http] ", 0)
		if captureHTTPInTrace {
			logger = log.New(&httpTraceWriter{runner: r}, "", 0)
		}
		options = append(options, option.WithDebugLog(logger))
	}
	r.client = anthropic.NewClient(options...)
	tools.Tracef = r.tracef
	return r
}

func (r *Runner) Run(ctx context.Context, prompt string) (Result, error) {
	r.trace = nil
	r.traceConversationHeader(prompt)
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for turn := 0; turn < r.maxTurns; turn++ {
		params := r.messageParams(messages, turn)
		if turn == 0 {
			r.tracef("[turn %d request] sending %d conversation message(s) to Anthropic with tool_choice=search", turn+1, len(messages))
		} else {
			r.tracef("[turn %d request] sending %d conversation message(s) to Anthropic", turn+1, len(messages))
		}
		message, err := r.client.Messages.New(ctx, params)
		if err != nil {
			r.tracef("[runner error] anthropic request failed: %v", err)
			return r.snapshotResult(""), err
		}

		messages = append(messages, message.ToParam())
		r.traceAssistantMessage(turn+1, message)
		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range message.Content {
			switch block := block.AsAny().(type) {
			case anthropic.ToolUseBlock:
				inputJSON, err := json.Marshal(block.Input)
				if err != nil {
					r.tracef("[runner error] marshal tool input for %s: %v", block.Name, err)
					return r.snapshotResult(""), err
				}
				r.tracef("[turn %d tool call %s] input\n%s", turn+1, block.Name, indentJSON(string(inputJSON)))
				output, isError, err := r.tools.Execute(ctx, block.Name, inputJSON)
				if err != nil {
					output = err.Error()
					isError = true
				}
				r.tracef("[turn %d tool result %s] is_error=%t\n%s", turn+1, block.Name, isError, indentJSON(output))
				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, output, isError))
			}
		}

		if len(toolResults) == 0 {
			return r.snapshotResult(extractText(message)), nil
		}

		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	err := fmt.Errorf("tool loop reached max turns (%d)", r.maxTurns)
	r.tracef("[runner error] %v", err)
	return r.snapshotResult(""), err
}

func (r *Runner) messageParams(messages []anthropic.MessageParam, turn int) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(r.model),
		MaxTokens: 1600,
		System:    r.systemBlocks(turn),
		Messages:  messages,
		Tools:     r.toolDefinitions(turn),
	}
	if turn == 0 {
		params.ToolChoice = anthropic.ToolChoiceParamOfTool("search")
	}
	return params
}

func (r *Runner) traceConversationHeader(prompt string) {
	if !r.verbose {
		return
	}
	r.trace = append(r.trace,
		"=== Conversation ===",
		"[initial system prompt]\n"+searchOnlySystemPrompt(),
		"[follow-up system prompt]\n"+fullSystemPrompt(),
		"[catalog snapshot]\n"+r.tools.Description(),
		fmt.Sprintf("[runner]\nmodel=%s max_turns=%d debug_http=%t", r.model, r.maxTurns, r.debugHTTP),
		"[user]\n"+prompt,
	)
}

func (r *Runner) systemBlocks(turn int) []anthropic.TextBlockParam {
	return []anthropic.TextBlockParam{
		{Text: r.systemPromptForTurn(turn)},
	}
}

func (r *Runner) toolDefinitions(turn int) []anthropic.ToolUnionParam {
	if turn == 0 {
		return []anthropic.ToolUnionParam{r.tools.SearchDefinition()}
	}
	return r.tools.Definitions()
}

func (r *Runner) systemPromptForTurn(turn int) string {
	if turn == 0 {
		return searchOnlySystemPrompt()
	}
	return fullSystemPrompt()
}

func (r *Runner) tracef(format string, args ...any) {
	if !r.verbose {
		return
	}
	r.trace = append(r.trace, fmt.Sprintf(format, args...))
}

func (r *Runner) traceHTTP(label, dump string) {
	if !r.verbose {
		return
	}
	r.trace = append(r.trace, formatHTTPTrace(label, dump))
}

func (r *Runner) traceAssistantMessage(turn int, message *anthropic.Message) {
	if !r.verbose {
		return
	}
	var parts []string
	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			parts = append(parts, block.Text)
		case anthropic.ToolUseBlock:
			inputJSON, _ := json.MarshalIndent(block.Input, "", "  ")
			parts = append(parts, fmt.Sprintf("tool_use %s\n%s", block.Name, string(inputJSON)))
		default:
			parts = append(parts, fmt.Sprintf("unsupported block %T", block))
		}
	}
	r.trace = append(r.trace, fmt.Sprintf("[assistant turn %d]\n%s", turn, strings.Join(parts, "\n\n")))
}

func extractText(message *anthropic.Message) string {
	parts := make([]string, 0, len(message.Content))
	for _, block := range message.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func searchOnlySystemPrompt() string {
	return strings.TrimSpace(`You are a helpful assistant. You have access to helpers that can help you answer the user's question. 
Use them if it helps you answer better. Use search to discover the relevant helpers.`)
}

func fullSystemPrompt() string {
	return strings.TrimSpace(`You are a helpful assistant. You have access to helpers that can help you answer the user's question. 
Use them if it helps you answer better.
Use search to discover the relevant helpers.

After search returns helper definitions, prefer a single execute(code) call that completes the full computation.
If one helper's output can be passed directly into another helper inside the same JavaScript snippet, 
do that instead of making multiple execute calls with intermediate results.

Also when you need certain parts of the response of one helper to decide how to call another helper, 
it's better to do that orchestration in a single execute call with JavaScript, rather than making multiple 
tool calls. For example if the helper returns a list of items and you want to call another helper on each item, it's better to do that 
iteration within the same JavaScript snippet. 

You have a ECMAScript 2023 environment to your disposal in the execute(code) helper, and you can use it to orchestrate 
calls to other helpers as needed.
`)
}

func indentJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "  <empty>"
	}
	var decoded any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		return "  " + strings.ReplaceAll(s, "\n", "\n  ")
	}
	b, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return "  " + strings.ReplaceAll(s, "\n", "\n  ")
	}
	return "  " + strings.ReplaceAll(string(b), "\n", "\n  ")
}

type httpTraceWriter struct {
	runner *Runner
}

func (w *httpTraceWriter) Write(p []byte) (int, error) {
	text := strings.TrimSpace(string(p))
	if text == "" || w.runner == nil {
		return len(p), nil
	}
	switch {
	case strings.HasPrefix(text, "Request Content:\n"):
		w.runner.traceHTTP("http request", strings.TrimPrefix(text, "Request Content:\n"))
	case strings.HasPrefix(text, "Response Content:\n"):
		w.runner.traceHTTP("http response", strings.TrimPrefix(text, "Response Content:\n"))
	default:
		w.runner.tracef("[http debug]\n%s", text)
	}
	return len(p), nil
}

func formatHTTPTrace(label, dump string) string {
	dump = strings.ReplaceAll(strings.TrimSpace(dump), "\r\n", "\n")
	headers := dump
	body := ""
	if parts := strings.SplitN(dump, "\n\n", 2); len(parts) == 2 {
		headers = strings.TrimSpace(parts[0])
		body = strings.TrimSpace(parts[1])
	}
	headers = sanitizeHTTPTraceHeaders(headers)
	var builder strings.Builder
	builder.WriteString("[")
	builder.WriteString(label)
	builder.WriteString("]\n")
	builder.WriteString(headers)
	builder.WriteString("\n\nbody\n")
	builder.WriteString(indentJSON(body))
	return builder.String()
}

func sanitizeHTTPTraceHeaders(headers string) string {
	lines := strings.Split(headers, "\n")
	for i, line := range lines {
		name, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "X-Api-Key") {
			lines[i] = name + ": <secure>"
		}
	}
	return strings.Join(lines, "\n")
}

var _ io.Writer = (*httpTraceWriter)(nil)
