package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gomcpexample/internal"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "weather-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: "Weather information server. Provides current weather data for locations using latitude/longitude coordinates or city/country codes.",
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_weather",
		Description: "Get current weather information for a location. You can provide either latitude/longitude coordinates OR city/country ISO-3166-1 alpha2 code. If you know the coordinates, provide them directly. If you only have the city name, you must provide the city name together with the country ISO-3166-1 alpha2 code.",
	}, handleWeatherTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: false,
	})

	addr := ":8080"
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting MCP HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server exited gracefully")
	}
}

type WeatherToolInput struct {
	Latitude  *float64 `json:"latitude,omitempty" jsonschema:"Latitude coordinate (optional if city is provided)"`
	Longitude *float64 `json:"longitude,omitempty" jsonschema:"Longitude coordinate (optional if city is provided)"`
	City      *string  `json:"city,omitempty" jsonschema:"City name (optional if latitude/longitude provided)"`
	Country   *string  `json:"country,omitempty" jsonschema:"Country ISO-3166-1 alpha2 code (optional if latitude/longitude provided, mandatory if city name is provided)"`
}

type WeatherToolOutput struct {
	Temperature        float64 `json:"temperature" jsonschema:"Temperature in Celsius"`
	WindSpeed          float64 `json:"wind_speed" jsonschema:"Wind speed in km/h"`
	WindDirection      int     `json:"wind_direction" jsonschema:"Wind direction in degrees"`
	WeatherDescription string  `json:"weather_description" jsonschema:"Human-readable weather description"`
	Time               string  `json:"time" jsonschema:"Time of the weather observation"`
}

func handleWeatherTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input WeatherToolInput,
) (*mcp.CallToolResult, WeatherToolOutput, error) {
	args := internal.WeatherFunctionArgs{
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		City:      input.City,
		Country:   input.Country,
	}

	result, err := internal.ExecuteWeatherFunction(args)
	if err != nil {
		return nil, WeatherToolOutput{}, err
	}

	output := WeatherToolOutput{
		Temperature:        result.Temperature,
		WindSpeed:          result.WindSpeed,
		WindDirection:      result.WindDirection,
		WeatherDescription: result.WeatherDescription,
		Time:               result.Time,
	}

	return nil, output, nil
}
