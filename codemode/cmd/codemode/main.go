package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	anthropicloop "github.com/preblog/codemode/internal/anthropic"
	"github.com/preblog/codemode/internal/app"
	"github.com/preblog/codemode/internal/config"
)

func main() {
	var (
		model      = flag.String("model", "claude-sonnet-4-6", "Anthropic model name")
		prompt     = flag.String("prompt", "", "One-shot prompt to run")
		maxTurns   = flag.Int("max-turns", 6, "Maximum Anthropic tool loop turns")
		timeoutSec = flag.Int("timeout-seconds", 10, "QuickJS evaluation timeout in seconds")
		memoryMB   = flag.Int("memory-mb", 32, "QuickJS memory limit in megabytes")
		verbose    = flag.Bool("verbose", false, "Print tool activity")
		noColor    = flag.Bool("no-color", false, "Disable ANSI colors in verbose output")
		saveTrace  = flag.String("save-trace", "", "Write the full conversation trace to a readable Markdown file")
		debugHTTP  = flag.Bool("debug-http", false, "Print raw Anthropic HTTP requests and responses to stderr, or capture them in -save-trace")
	)
	flag.Parse()

	tracePath := strings.TrimSpace(*saveTrace)
	captureTrace := *verbose || tracePath != ""
	captureHTTPInTrace := *debugHTTP && tracePath != ""

	// captureTrace also enables internal verbose logging so traces can be collected for -save-trace
	cfg, err := config.Load(config.Options{
		Model:         *model,
		MaxTurns:      *maxTurns,
		EvalTimeoutS:  *timeoutSec,
		MemoryLimitMB: *memoryMB,
		Verbose:       captureTrace,
		DebugHTTP:     *debugHTTP,
	})
	if err != nil {
		log.Fatal(err)
	}

	runner, err := app.New(cfg, captureHTTPInTrace)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := runner.Close(); err != nil {
			log.Printf("close runner: %v", err)
		}
	}()

	input := strings.TrimSpace(*prompt)
	if input == "" {
		input = strings.TrimSpace(strings.Join(flag.Args(), " "))
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "Provide a prompt with -prompt or as positional arguments.")
		os.Exit(2)
	}

	result, err := runner.Run(context.Background(), input)
	if artifactErr := persistRunArtifacts(tracePath, input, cfg, result, err, *verbose, !*noColor, captureHTTPInTrace); artifactErr != nil {
		log.Fatal(artifactErr)
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
}

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiCyan   = "\x1b[36m"
	ansiBlue   = "\x1b[94m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

func renderVerboseTrace(lines []string, color bool) {
	fmt.Print(renderVerboseTraceText(lines, color))
}

func filterTraceForConsole(lines []string, keepHTTP bool) []string {
	if keepHTTP {
		return lines
	}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "[http request]") || strings.HasPrefix(line, "[http response]") {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func renderVerboseTraceText(lines []string, color bool) string {
	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(styleVerboseLine(line, color))
		builder.WriteByte('\n')
		if strings.HasPrefix(line, "[") || strings.HasPrefix(line, "===") {
			separator := strings.Repeat("-", 72)
			if color {
				separator = ansiDim + separator + ansiReset
			}
			builder.WriteString(separator)
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func styleVerboseLine(line string, color bool) string {
	if !color {
		return line
	}
	switch {
	case strings.HasPrefix(line, "==="):
		return ansiBlue + line + ansiReset
	case strings.HasPrefix(line, "[initial system prompt]"), strings.HasPrefix(line, "[follow-up system prompt]"):
		return ansiCyan + line + ansiReset
	case strings.HasPrefix(line, "[api description]"):
		return ansiYellow + line + ansiReset
	case strings.HasPrefix(line, "[runner]"):
		return ansiDim + line + ansiReset
	case strings.HasPrefix(line, "[user]"):
		return ansiGreen + line + ansiReset
	case strings.HasPrefix(line, "[assistant"):
		return ansiBlue + line + ansiReset
	case strings.Contains(line, "tool result") && strings.Contains(line, "is_error=true"):
		return ansiRed + line + ansiReset
	case strings.Contains(line, "tool call") || strings.Contains(line, "tool result"):
		return ansiYellow + line + ansiReset
	default:
		return line
	}
}

func persistRunArtifacts(tracePath, prompt string, cfg config.Config, result anthropicloop.Result, runErr error, verbose bool, color bool, captureHTTPInTrace bool) error {
	if tracePath != "" {
		if err := saveTraceFile(tracePath, prompt, cfg, traceFinalText(result, runErr), result.Trace); err != nil {
			return err
		}
	}
	if verbose && len(result.Trace) > 0 {
		renderVerboseTrace(filterTraceForConsole(result.Trace, !captureHTTPInTrace), color)
		fmt.Println()
	}
	return nil
}

func traceFinalText(result anthropicloop.Result, runErr error) string {
	if strings.TrimSpace(result.Text) != "" {
		return result.Text
	}
	if runErr != nil {
		return "ERROR: " + runErr.Error()
	}
	return ""
}

func saveTraceFile(path, prompt string, cfg config.Config, finalText string, trace []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create trace directory: %w", err)
	}
	content := buildTraceDocument(prompt, cfg, finalText, trace)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write trace file: %w", err)
	}
	return nil
}

func buildTraceDocument(prompt string, cfg config.Config, finalText string, trace []string) string {
	var builder strings.Builder
	builder.WriteString("# CodeMode Trace\n\n")
	builder.WriteString("## Run Metadata\n\n")
	fmt.Fprintf(&builder, "- Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&builder, "- Model: %s\n", cfg.Model)
	fmt.Fprintf(&builder, "- Max turns: %d\n", cfg.MaxTurns)
	fmt.Fprintf(&builder, "- Debug HTTP: %t\n", cfg.DebugHTTP)
	builder.WriteString("- MCP server: embedded demo\n")
	builder.WriteString("\n## Prompt\n\n```text\n")
	builder.WriteString(prompt)
	builder.WriteString("\n```\n\n")
	builder.WriteString("## Final Answer\n\n```markdown\n")
	builder.WriteString(finalText)
	builder.WriteString("\n```\n\n")
	builder.WriteString("## Transcript\n\n```text\n")
	builder.WriteString(renderVerboseTraceText(trace, false))
	builder.WriteString("```\n")
	return builder.String()
}
