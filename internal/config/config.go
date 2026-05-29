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

type Config struct {
	Feeds []FeedConfig `yaml:"feeds"`
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
