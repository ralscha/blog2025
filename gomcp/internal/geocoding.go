package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const geocodingBaseURL = "https://geocoding-api.open-meteo.com/v1/search"

type geocodingResult struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
}

type geocodingResponse struct {
	Results []geocodingResult `json:"results"`
}

func getLatLng(city, country string) (*geocodingResult, error) {

	params := url.Values{}
	params.Add("name", city)
	params.Add("count", "1")
	params.Add("language", "en")
	params.Add("format", "json")
	params.Add("countryCode", country)

	fullURL := fmt.Sprintf("%s?%s", geocodingBaseURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make geocoding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var geoResp geocodingResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	if len(geoResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for location: %s, %s", city, country)
	}

	return &geoResp.Results[0], nil
}
