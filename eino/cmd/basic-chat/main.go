package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"einoexamples/internal/shared"

	"github.com/cloudwego/eino/schema"
)

func main() {
	prompt := flag.String("prompt", "What is Eino in one paragraph?", "Prompt to send to the model")
	flag.Parse()

	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("You are a concise assistant."),
		schema.UserMessage(strings.TrimSpace(*prompt)),
	}

	reply, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(reply.Content)
}
