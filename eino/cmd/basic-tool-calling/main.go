package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"einoexamples/internal/shared"

	"github.com/cloudwego/eino/adk"
	toolcomp "github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type weatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"The city to look up"`
}

type weatherOutput struct {
	Forecast string `json:"forecast"`
}

func main() {
	question := flag.String("question", "Should I take an umbrella in Hangzhou today?", "Question for the tool-using agent")
	flag.Parse()

	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	weatherTool, err := toolutils.InferTool("lookup_weather", "Look up the current weather for a city.", lookupWeather)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "weather-agent",
		Description: "An assistant that can answer questions and call a weather tool.",
		Instruction: "You are a helpful assistant. Use tools when needed.",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []toolcomp.BaseTool{weatherTool},
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

func lookupWeather(_ context.Context, input *weatherInput) (*weatherOutput, error) {
	forecasts := map[string]string{
		"hangzhou": "Light rain, 21C",
		"beijing":  "Sunny, 24C",
		"shanghai": "Cloudy, 23C",
	}

	city := strings.TrimSpace(input.City)
	forecast, ok := forecasts[strings.ToLower(city)]
	if !ok {
		forecast = "Weather data unavailable, assume mild conditions."
	}

	return &weatherOutput{
		Forecast: fmt.Sprintf("%s: %s", city, forecast),
	}, nil
}
