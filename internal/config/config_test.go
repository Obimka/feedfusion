package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	// Test that LoadConfig returns empty config when file doesn't exist
	cfg, err := LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected config to be non-nil")
	}
	if len(cfg.Feeds) != 0 {
		t.Errorf("Expected empty feeds, got: %d", len(cfg.Feeds))
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
feeds:
  - url: https://example.com/feed.xml
    title: Example Feed
    include: true
  - url: https://test.com/rss
    title: Test Feed
    include: false
weather_cities:
  - name: Paris
    lat: 48.8566
    lon: 2.3522
    timezone: Europe/Paris
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Feeds) != 2 {
		t.Errorf("Expected 2 feeds, got: %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].URL != "https://example.com/feed.xml" {
		t.Errorf("Expected feed URL to be https://example.com/feed.xml, got: %s", cfg.Feeds[0].URL)
	}
	if cfg.Feeds[0].Title != "Example Feed" {
		t.Errorf("Expected feed title to be Example Feed, got: %s", cfg.Feeds[0].Title)
	}
	if !cfg.Feeds[0].Include {
		t.Error("Expected first feed to be included")
	}

	if len(cfg.WeatherCities) != 1 {
		t.Errorf("Expected 1 weather city, got: %d", len(cfg.WeatherCities))
	}
	if cfg.WeatherCities[0].Name != "Paris" {
		t.Errorf("Expected city name to be Paris, got: %s", cfg.WeatherCities[0].Name)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.yaml")

	invalidContent := `
feeds:
  - url: https://example.com
    title: Example
  - invalid yaml here
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
