package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"einoexamples/internal/shared"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	question := flag.String("question", "Use the Playwright MCP tools to open https://example.com, then tell me the page title and the main heading.", "Question for the Playwright MCP agent")
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
		Name:    "eino-playwright-example",
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
		Name:        "playwright-mcp-agent",
		Description: "An agent that can browse web pages through Playwright MCP tools running in Docker.",
		Instruction: strings.Join([]string{
			"You are a browser automation assistant.",
			"Use the Playwright MCP tools to inspect and interact with pages when the task requires browsing.",
			"Prefer the smallest set of actions needed to answer accurately.",
		}, " "),
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: mcpTools,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	prompt := strings.TrimSpace(*question)
	if _, err := shared.PrintQueryAgentEvents(prompt, runner.Query(ctx, prompt)); err != nil {
		log.Fatal(err)
	}
}
