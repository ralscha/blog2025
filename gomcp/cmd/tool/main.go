package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"gomcpexample/internal"

	"github.com/joho/godotenv"
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

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create Gen AI client: %v", err)
	}

	weatherFunc := &genai.FunctionDeclaration{
		Name:        "get_weather",
		Description: "Get current weather information for a location. You can provide either latitude/longitude coordinates OR city/country ISO-3166-1 alpha2 code. If you know the coordinates, provide them directly. If you only have the city name, you must provide the city name together with the country ISO-3166-1 alpha2 code.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"latitude": {
					Type:        genai.TypeNumber,
					Description: "Latitude coordinate (optional if city is provided)",
				},
				"longitude": {
					Type:        genai.TypeNumber,
					Description: "Longitude coordinate (optional if city is provided)",
				},
				"city": {
					Type:        genai.TypeString,
					Description: "City name (optional if latitude/longitude provided)",
				},
				"country": {
					Type:        genai.TypeString,
					Description: "Country ISO-3166-1 alpha2 code (optional if latitude/longitude provided, mandatory if city name is provided)",
				},
			},
		},
	}

	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{
			FunctionDeclarations: []*genai.FunctionDeclaration{weatherFunc},
		}},
		Temperature: genai.Ptr[float32](0.3),
	}

	userPrompts := []string{"Where is it warmer right now, Rome or Athens?",
		"What's the weather like in Wellington, New Zealand?",
		"What is tool calling? Give me a two sentence summary."}

	for _, userPrompt := range userPrompts {
		fmt.Println("User:", userPrompt)
		userMessage := genai.Text(userPrompt)

		result, err := client.Models.GenerateContent(
			ctx,
			GEMINI_MODEL,
			userMessage,
			config,
		)
		if err != nil {
			log.Printf("Error generating content: %v", err)
			continue
		}

		if err := processResponse(ctx, client, userMessage, result, config); err != nil {
			log.Printf("Error processing response: %v", err)
		}

		fmt.Println("==============================")
	}

}

func processResponse(ctx context.Context, client *genai.Client, userMessage []*genai.Content, result *genai.GenerateContentResponse, config *genai.GenerateContentConfig) error {
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

					if part.FunctionCall.Name != "get_weather" {
						fmt.Println("Unknown function call:", part.FunctionCall.Name)
						continue
					}

					args := internal.WeatherFunctionArgs{}
					if lat, ok := part.FunctionCall.Args["latitude"].(float64); ok {
						args.Latitude = &lat
					}
					if lon, ok := part.FunctionCall.Args["longitude"].(float64); ok {
						args.Longitude = &lon
					}
					if city, ok := part.FunctionCall.Args["city"].(string); ok {
						args.City = &city
					}
					if country, ok := part.FunctionCall.Args["country"].(string); ok {
						args.Country = &country
					}

					weatherData, err := internal.ExecuteWeatherFunction(args)
					if err != nil {
						return fmt.Errorf("function execution failed: %w", err)
					}

					weatherResponse := map[string]any{
						"temperature":         weatherData.Temperature,
						"wind_speed":          weatherData.WindSpeed,
						"wind_direction":      weatherData.WindDirection,
						"weather_description": weatherData.WeatherDescription,
						"time":                weatherData.Time,
					}

					functionResponseParts = append(functionResponseParts, &genai.Part{
						FunctionResponse: &genai.FunctionResponse{
							Name:     part.FunctionCall.Name,
							Response: weatherResponse,
						},
					})
				}

				conversationHistory = append(conversationHistory, candidate.Content)
				conversationHistory = append(conversationHistory, &genai.Content{
					Parts: functionResponseParts,
				})

				nextResult, err := client.Models.GenerateContent(
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
