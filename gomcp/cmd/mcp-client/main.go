package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "weather-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint: "http://localhost:8080",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	demonstrateListTools(ctx, session)
	demonstrateToolsIterator(ctx, session)
	demonstrateCallTool(ctx, session)
	demonstrateListResources(ctx, session)
	demonstrateListPrompts(ctx, session)
}

func demonstrateListTools(ctx context.Context, session *mcp.ClientSession) {
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Printf("Failed to list tools: %v", err)
		return
	}

	fmt.Printf("Found %d tool(s):\n", len(result.Tools))
	for i, tool := range result.Tools {
		fmt.Printf("  %d. Name: %s\n", i+1, tool.Name)
		fmt.Printf("     Description: %s\n", tool.Description)
		if tool.InputSchema != nil {
			schema, _ := json.MarshalIndent(tool.InputSchema, "     ", "  ")
			fmt.Printf("     Input Schema: %s\n", string(schema))
		}
	}
}

func demonstrateToolsIterator(ctx context.Context, session *mcp.ClientSession) {
	count := 0
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			log.Printf("Iterator error: %v", err)
			return
		}
		count++
		fmt.Printf("  Tool #%d: %s\n", count, tool.Name)
	}
}

func demonstrateCallTool(ctx context.Context, session *mcp.ClientSession) {
	result1, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_weather",
		Arguments: map[string]any{
			"latitude":  47.3769,
			"longitude": 8.5417,
		},
	})
	if err != nil {
		log.Printf("Failed to call tool: %v", err)
	} else {
		displayToolResult(result1)
	}

	result2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_weather",
		Arguments: map[string]any{
			"city":    "Paris",
			"country": "FR",
		},
	})
	if err != nil {
		log.Printf("Failed to call tool: %v", err)
	} else {
		displayToolResult(result2)
	}

}

func displayToolResult(result *mcp.CallToolResult) {
	if result.IsError {
		fmt.Println("Tool returned error")
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				fmt.Printf("     Error: %s\n", text.Text)
			}
		}
		return
	}

	fmt.Println("Tool call successful")

	if result.StructuredContent != nil {
		data, err := json.MarshalIndent(result.StructuredContent, "     ", "  ")
		if err == nil {
			fmt.Printf("     Result: %s\n", string(data))
		}
	}

	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			fmt.Printf("     %s\n", text.Text)
		}
	}
}

func demonstrateListResources(ctx context.Context, session *mcp.ClientSession) {
	result, err := session.ListResources(ctx, nil)
	if err != nil {
		log.Printf("Failed to list resources: %v", err)
		return
	}

	if len(result.Resources) == 0 {
		fmt.Println("No resources available on this server")
	} else {
		fmt.Printf("Found %d resource(s):\n", len(result.Resources))
		for i, resource := range result.Resources {
			fmt.Printf("  %d. Name: %s\n", i+1, resource.Name)
			fmt.Printf("     URI: %s\n", resource.URI)
			fmt.Printf("     Description: %s\n", resource.Description)
			if resource.MIMEType != "" {
				fmt.Printf("     MIME Type: %s\n", resource.MIMEType)
			}
		}

		if len(result.Resources) > 0 {
			fmt.Println("\n  Demonstrating ReadResource:")
			readResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
				URI: result.Resources[0].URI,
			})
			if err != nil {
				log.Printf("  Failed to read resource: %v", err)
			} else {
				fmt.Printf("  Resource contents:\n")
				for _, content := range readResult.Contents {
					if content.Text != "" {
						fmt.Printf("    %s\n", content.Text)
					}
				}
			}
		}
	}
	fmt.Println()
}

func demonstrateListPrompts(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("=== Demonstrating ListPrompts ===")

	result, err := session.ListPrompts(ctx, nil)
	if err != nil {
		log.Printf("Failed to list prompts: %v", err)
		return
	}

	if len(result.Prompts) == 0 {
		fmt.Println("No prompts available on this server")
	} else {
		fmt.Printf("Found %d prompt(s):\n", len(result.Prompts))
		for i, prompt := range result.Prompts {
			fmt.Printf("  %d. Name: %s\n", i+1, prompt.Name)
			fmt.Printf("     Description: %s\n", prompt.Description)
			if len(prompt.Arguments) > 0 {
				fmt.Printf("     Arguments:\n")
				for _, arg := range prompt.Arguments {
					required := ""
					if arg.Required {
						required = " (required)"
					}
					fmt.Printf("       - %s: %s%s\n", arg.Name, arg.Description, required)
				}
			}
		}

		if len(result.Prompts) > 0 {
			fmt.Println("\n  Demonstrating GetPrompt:")
			promptResult, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      result.Prompts[0].Name,
				Arguments: map[string]string{},
			})
			if err != nil {
				log.Printf("  Failed to get prompt: %v", err)
			} else {
				fmt.Printf("  Prompt: %s\n", promptResult.Description)
				fmt.Printf("  Messages:\n")
				for _, msg := range promptResult.Messages {
					fmt.Printf("    Role: %s\n", msg.Role)
					if text, ok := msg.Content.(*mcp.TextContent); ok {
						fmt.Printf("    Content: %s\n", text.Text)
					}
				}
			}
		}
	}
	fmt.Println()
}
