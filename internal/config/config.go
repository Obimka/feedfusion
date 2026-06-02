package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FeedConfig struct {
	URL     string `yaml:"url"`
	Title   string `yaml:"title"`
	Include bool   `yaml:"include"`
}

type WeatherCity struct {
	Name     string  `yaml:"name"`
	Lat      float64 `yaml:"lat"`
	Lon      float64 `yaml:"lon"`
	Timezone string  `yaml:"timezone"`
}

type Config struct {
	ServerPort      int     `yaml:"server_port"`
	MistralAPIKey   string  `yaml:"mistral_api_key"`
	AllowRegistration bool   `yaml:"allow_registration"`
	Feeds          []FeedConfig  `yaml:"feeds"`
	WeatherCities  []WeatherCity `yaml:"weather_cities"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist, return empty config
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}
