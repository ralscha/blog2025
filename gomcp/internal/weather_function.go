package internal

import (
	"fmt"
)

type WeatherFunctionArgs struct {
	Latitude  *float64
	Longitude *float64
	City      *string
	Country   *string
}

type WeatherFunctionResult struct {
	Temperature        float64
	WindSpeed          float64
	WindDirection      int
	WeatherDescription string
	Time               string
}

func ExecuteWeatherFunction(args WeatherFunctionArgs) (*WeatherFunctionResult, error) {
	var latitude, longitude float64
	var hasCoords bool

	if args.Latitude != nil && args.Longitude != nil {
		latitude = *args.Latitude
		longitude = *args.Longitude
		hasCoords = true
	}

	if !hasCoords {
		city, hasCity := args.City, args.City != nil
		country := args.Country

		if !hasCity {
			return nil, fmt.Errorf("either latitude/longitude or city must be provided")
		}
		geoResult, err := getLatLng(*city, *country)
		if err != nil {
			return nil, fmt.Errorf("geocoding failed: %w", err)
		}

		latitude = geoResult.Latitude
		longitude = geoResult.Longitude
	}

	weatherInfo, err := getCurrentWeather(latitude, longitude)
	if err != nil {
		return nil, fmt.Errorf("weather fetch failed: %w", err)
	}

	return &WeatherFunctionResult{
		Temperature:        weatherInfo.Temperature,
		WindSpeed:          weatherInfo.WindSpeed,
		WindDirection:      weatherInfo.WindDirection,
		WeatherDescription: weatherInfo.WeatherCodeText,
		Time:               weatherInfo.Time,
	}, nil
}
