package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/genai"
)

const GEMINI_MODEL = "models/gemini-flash-lite-latest"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	ctx := context.Background()

	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create Gen AI client: %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "playwright-client",
		Version: "1.0.0",
	}, nil)

	cmd := exec.Command("docker", "run", "-i", "--rm", "--init", "--pull=always", "mcr.microsoft.com/playwright/mcp")
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect to Playwright MCP server: %v", err)
	}
	defer session.Close()

	time.Sleep(2 * time.Second)

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	if len(toolsResult.Tools) == 0 {
		log.Fatal("No tools available on Playwright MCP server")
	}

	fmt.Println("Available Playwright tools:")
	for _, tool := range toolsResult.Tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	var functionDeclarations []*genai.FunctionDeclaration
	for _, tool := range toolsResult.Tools {
		funcDecl := convertMCPToolToGeminiFunction(*tool)
		functionDeclarations = append(functionDeclarations, funcDecl)
	}

	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{
			FunctionDeclarations: functionDeclarations,
		}},
		Temperature: genai.Ptr[float32](0.3),
	}

	userPrompts := []string{
		"Navigate to https://github.com/trending and tell me the top 3 trending repositories today",
		"Visit https://news.ycombinator.com and give me the title of the top story",
	}

	for _, userPrompt := range userPrompts {
		fmt.Println("User:", userPrompt)
		userMessage := genai.Text(userPrompt)

		result, err := genaiClient.Models.GenerateContent(
			ctx,
			GEMINI_MODEL,
			userMessage,
			config,
		)
		if err != nil {
			log.Printf("Error generating content: %v", err)
			continue
		}

		if err := processResponse(ctx, genaiClient, session, userMessage, result, config); err != nil {
			log.Printf("Error processing response: %v", err)
		}
		fmt.Println()
	}
}

func convertMCPToolToGeminiFunction(tool mcp.Tool) *genai.FunctionDeclaration {
	funcDecl := &genai.FunctionDeclaration{
		Name:        tool.Name,
		Description: tool.Description,
	}

	if tool.InputSchema != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			funcDecl.Parameters = convertJSONSchemaToGemini(schema)
		}
	}

	return funcDecl
}

func convertJSONSchemaToGemini(schema map[string]any) *genai.Schema {
	geminiSchema := &genai.Schema{}

	if typeStr, ok := schema["type"].(string); ok {
		switch typeStr {
		case "object":
			geminiSchema.Type = genai.TypeObject
		case "string":
			geminiSchema.Type = genai.TypeString
		case "number":
			geminiSchema.Type = genai.TypeNumber
		case "integer":
			geminiSchema.Type = genai.TypeInteger
		case "boolean":
			geminiSchema.Type = genai.TypeBoolean
		case "array":
			geminiSchema.Type = genai.TypeArray
		}
	}

	if desc, ok := schema["description"].(string); ok {
		geminiSchema.Description = desc
	}

	if properties, ok := schema["properties"].(map[string]any); ok {
		geminiSchema.Properties = make(map[string]*genai.Schema)
		for propName, propSchema := range properties {
			if propSchemaMap, ok := propSchema.(map[string]any); ok {
				geminiSchema.Properties[propName] = convertJSONSchemaToGemini(propSchemaMap)
			}
		}
	}

	if required, ok := schema["required"].([]any); ok {
		geminiSchema.Required = make([]string, len(required))
		for i, req := range required {
			if reqStr, ok := req.(string); ok {
				geminiSchema.Required[i] = reqStr
			}
		}
	}

	if items, ok := schema["items"].(map[string]any); ok {
		geminiSchema.Items = convertJSONSchemaToGemini(items)
	}

	return geminiSchema
}

func processResponse(
	ctx context.Context,
	genaiClient *genai.Client,
	mcpSession *mcp.ClientSession,
	userMessage []*genai.Content,
	result *genai.GenerateContentResponse,
	config *genai.GenerateContentConfig,
) error {
	const maxLoops = 3
	conversationHistory := slices.Clone(userMessage)
	currentResult := result

	for loop := range maxLoops {
		hasToolCalls := false

		for _, candidate := range currentResult.Candidates {
			if candidate.Content == nil {
				continue
			}

			var functionCalls []*genai.Part
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					functionCalls = append(functionCalls, part)
				}
			}

			if len(functionCalls) > 0 {
				hasToolCalls = true
				var functionResponseParts []*genai.Part

				for _, part := range functionCalls {
					fmt.Printf("Calling tool (loop %d): %s\n", loop+1, part.FunctionCall.Name)

					toolResult, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					})
					if err != nil {
						return fmt.Errorf("MCP tool call failed: %w", err)
					}

					var responseData map[string]any
					if toolResult.StructuredContent != nil {
						if structured, ok := toolResult.StructuredContent.(map[string]any); ok {
							responseData = structured
						}
					} else {
						responseData = make(map[string]any)
						for _, content := range toolResult.Content {
							if text, ok := content.(*mcp.TextContent); ok {
								responseData["result"] = text.Text
							}
						}
					}

					functionResponseParts = append(functionResponseParts, &genai.Part{
						FunctionResponse: &genai.FunctionResponse{
							Name:     part.FunctionCall.Name,
							Response: responseData,
						},
					})
				}

				conversationHistory = append(conversationHistory, candidate.Content)
				conversationHistory = append(conversationHistory, &genai.Content{
					Parts: functionResponseParts,
				})

				nextResult, err := genaiClient.Models.GenerateContent(
					ctx,
					GEMINI_MODEL,
					conversationHistory,
					config,
				)
				if err != nil {
					return fmt.Errorf("generation failed at loop %d: %w", loop+1, err)
				}

				currentResult = nextResult
				break
			}
		}

		if !hasToolCalls {
			fmt.Printf("Assistant: %s\n", currentResult.Text())
			return nil
		}
	}

	fmt.Printf("Assistant: %s\n", currentResult.Text())
	return nil
}
