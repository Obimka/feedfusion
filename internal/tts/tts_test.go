package tts

import (
	"testing"
)

func TestGenerateSpeech_EmptyText(t *testing.T) {
	_, err := GenerateSpeech("", "c69964a6-ab8b-4f8a-9465-ec0925096ec8", "voxtral-mini-tts-2603", "test-key", "https://api.mistral.ai/v1/audio/")
	if err == nil {
		t.Error("Expected error for empty text")
	}
}

func TestGenerateSpeech_EmptyAPIKey(t *testing.T) {
	_, err := GenerateSpeech("Hello", "c69964a6-ab8b-4f8a-9465-ec0925096ec8", "voxtral-mini-tts-2603", "", "https://api.mistral.ai/v1/audio/")
	if err == nil {
		t.Error("Expected error for empty API key")
	}
}

func TestDefaultTTSConfig(t *testing.T) {
	config := DefaultTTSConfig()

	if config.APIBaseURL != "https://api.mistral.ai/v1/audio/" {
		t.Errorf("Expected APIBaseURL to be https://api.mistral.ai/v1/audio/, got %s", config.APIBaseURL)
	}

	if config.VoiceID != "c69964a6-ab8b-4f8a-9465-ec0925096ec8" {
		t.Errorf("Expected VoiceID to be c69964a6-ab8b-4f8a-9465-ec0925096ec8, got %s", config.VoiceID)
	}

	if config.Model != "voxtral-mini-tts-2603" {
		t.Errorf("Expected Model to be voxtral-mini-tts-2603, got %s", config.Model)
	}

	if config.APIKey != "" {
		t.Error("Expected APIKey to be empty by default")
	}
}

func TestAvailableVoices(t *testing.T) {
	if len(AvailableVoices) == 0 {
		t.Error("Expected AvailableVoices to have at least one voice")
	}

	// Check that default voice ID is available
	found := false
	for _, v := range AvailableVoices {
		if v == "c69964a6-ab8b-4f8a-9465-ec0925096ec8" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected default voice ID to be in AvailableVoices")
	}
}

func TestAvailableModels(t *testing.T) {
	if len(AvailableModels) == 0 {
		t.Error("Expected AvailableModels to have at least one model")
	}

	// Check that voxtral-mini-tts-2603 is available
	found := false
	for _, m := range AvailableModels {
		if m == "voxtral-mini-tts-2603" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected voxtral-mini-tts-2603 to be in AvailableModels")
	}
}
