package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const forecastBaseURL = "https://api.open-meteo.com/v1/forecast"

type currentWeather struct {
	Temperature     float64 `json:"temperature_2m"`
	WindSpeed       float64 `json:"windspeed_10m"`
	WindDirection   int     `json:"winddirection_10m"`
	WeatherCode     int     `json:"weathercode"`
	WeatherCodeText string  `json:"-"`
	Time            string  `json:"time"`
}

type weatherResponse struct {
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Timezone  string         `json:"timezone"`
	Current   currentWeather `json:"current"`
}

func getCurrentWeather(latitude, longitude float64) (*currentWeather, error) {

	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%.6f", latitude))
	params.Add("longitude", fmt.Sprintf("%.6f", longitude))
	params.Add("current", "temperature_2m,windspeed_10m,winddirection_10m,weathercode")

	fullURL := fmt.Sprintf("%s?%s", forecastBaseURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make weather request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var weatherResp weatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return nil, fmt.Errorf("failed to parse weather response: %w", err)
	}
	weatherResp.Current.WeatherCodeText = getWeatherDescription(weatherResp.Current.WeatherCode)
	return &weatherResp.Current, nil
}

func getWeatherDescription(code int) string {
	descriptions := map[int]string{
		0:  "Clear sky",
		1:  "Mainly clear",
		2:  "Partly cloudy",
		3:  "Overcast",
		45: "Foggy",
		48: "Depositing rime fog",
		51: "Light drizzle",
		53: "Moderate drizzle",
		55: "Dense drizzle",
		56: "Light freezing drizzle",
		57: "Dense freezing drizzle",
		61: "Slight rain",
		63: "Moderate rain",
		65: "Heavy rain",
		66: "Light freezing rain",
		67: "Heavy freezing rain",
		71: "Slight snow fall",
		73: "Moderate snow fall",
		75: "Heavy snow fall",
		77: "Snow grains",
		80: "Slight rain showers",
		81: "Moderate rain showers",
		82: "Violent rain showers",
		85: "Slight snow showers",
		86: "Heavy snow showers",
		95: "Thunderstorm",
		96: "Thunderstorm with slight hail",
		99: "Thunderstorm with heavy hail",
	}

	if desc, ok := descriptions[code]; ok {
		return desc
	}
	return fmt.Sprintf("Unknown (code: %d)", code)
}
