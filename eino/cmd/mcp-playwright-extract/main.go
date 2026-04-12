package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"einoexamples/internal/shared"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/adk"
	toolcomp "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type extractedPage struct {
	URL         string          `json:"url"`
	Title       string          `json:"title"`
	MainHeading string          `json:"main_heading"`
	Summary     string          `json:"summary"`
	Links       []extractedLink `json:"links"`
}

type extractedLink struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

func main() {
	pageURL := flag.String("url", "https://example.com", "Page to extract structured data from")
	image := flag.String("docker-image", "mcr.microsoft.com/playwright/mcp", "Docker image that runs the Playwright MCP server")
	flag.Parse()

	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	mcpClient, err := client.NewStdioMCPClient(
		"docker",
		nil,
		"run",
		"-i",
		"--rm",
		"--init",
		"--pull=always",
		strings.TrimSpace(*image),
	)
	if err != nil {
		log.Fatalf("create MCP client: %v", err)
	}
	defer func() {
		_ = mcpClient.Close()
	}()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "eino-playwright-extract-example",
		Version: "1.0.0",
	}

	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("initialize MCP client: %v", err)
	}

	fmt.Printf("Connected to MCP server: %s %s\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)

	mcpTools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: mcpClient})
	if err != nil {
		log.Fatalf("load MCP tools: %v", err)
	}
	if len(mcpTools) == 0 {
		log.Fatal("the Playwright MCP server exposed no tools")
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "playwright-mcp-extract-agent",
		Description: "An agent that can visit a page with Playwright MCP and return structured JSON.",
		Instruction: strings.Join([]string{
			"You are a browser extraction assistant.",
			"Use the Playwright MCP tools to inspect the live page before answering.",
			"Return only valid JSON with this exact shape:",
			`{"url":"string","title":"string","main_heading":"string","summary":"string","links":[{"text":"string","href":"string"}]}`,
			"Limit links to the 5 most relevant visible links on the page.",
			"Do not wrap the JSON in markdown fences.",
		}, " "),
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toBaseTools(mcpTools),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	prompt := fmt.Sprintf("Open %s with the Playwright MCP tools and extract the requested structured data.", strings.TrimSpace(*pageURL))
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	responseText, err := shared.PrintQueryAgentEvents(prompt, runner.Query(ctx, prompt))
	if err != nil {
		log.Fatal(err)
	}

	parsed, err := decodeExtractedPage(responseText)
	if err != nil {
		log.Fatalf("parse structured JSON response: %v", err)
	}

	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		log.Fatalf("format structured JSON response: %v", err)
	}

	fmt.Println("\n[structured result]")
	fmt.Println(string(pretty))
}

func decodeExtractedPage(raw string) (*extractedPage, error) {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON object found in response: %q", trimmed)
	}

	trimmed = trimmed[start : end+1]

	var page extractedPage
	if err := json.Unmarshal([]byte(trimmed), &page); err != nil {
		return nil, err
	}

	return &page, nil
}

func toBaseTools(tools []toolcomp.BaseTool) []toolcomp.BaseTool {
	return tools
}
