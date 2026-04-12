package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"einoexamples/internal/shared"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "chat-agent",
		Description: "A minimal multi-turn assistant.",
		Instruction: "You are a helpful assistant that answers clearly and briefly.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	history := make([]*schema.Message, 0, 16)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Ask something. Submit an empty line to exit.")
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		history = append(history, schema.UserMessage(line))
		assistantReply, err := shared.PrintAgentEvents(runner.Run(ctx, history))
		if err != nil {
			log.Fatal(err)
		}
		history = append(history, schema.AssistantMessage(assistantReply, nil))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
