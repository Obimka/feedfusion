package tts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TTSConfig holds configuration for the TTS service
type TTSConfig struct {
	APIKey      string
	APIBaseURL  string
	VoiceID     string // Mistral voice ID (e.g., "c69964a6-ab8b-4f8a-9465-ec0925096ec8")
	Model       string // e.g., "voxtral-mini-tts-2603"
}

// DefaultTTSConfig returns default TTS configuration
func DefaultTTSConfig() TTSConfig {
	return TTSConfig{
		APIKey:     "", // Must be set by user
		APIBaseURL:  "https://api.mistral.ai/v1/audio/",
		VoiceID:    "c69964a6-ab8b-4f8a-9465-ec0925096ec8", // Paul - Neutral (en_us)
		Model:      "voxtral-mini-tts-2603",
	}
}

// MistralTTSRequest represents a request to the Mistral TTS API
type MistralTTSRequest struct {
	Model       string `json:"model"`
	Input       string `json:"input"`
	VoiceID     string `json:"voice_id"`
	Format      string `json:"response_format,omitempty"`
}

// MistralTTSResponse represents the response from the Mistral TTS API
type MistralTTSResponse struct {
	AudioData string `json:"audio_data"`
}

// GenerateSpeech generates speech from text using Mistral TTS API
// Returns audio data in WAV format or error
func GenerateSpeech(text, voice, model, apiKey, apiBaseURL string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Use voice as voice_id if provided, otherwise use default from config
	voiceID := voice
	if voiceID == "" {
		voiceID = "c69964a6-ab8b-4f8a-9465-ec0925096ec8" // Default: Paul - Neutral (en_us)
	}

	// Use model if provided, otherwise use default
	if model == "" {
		model = "voxtral-mini-tts-2603"
	}

	// Create request for Mistral API
	request := MistralTTSRequest{
		Model:   model,
		Input:   text,
		VoiceID: voiceID,
		Format:  "wav", // Request WAV format
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request - Mistral uses /v1/audio/speech endpoint
	url := "https://api.mistral.ai/v1/audio/speech"
	if apiBaseURL != "" && apiBaseURL != "https://api.mistral.ai/v1/audio/" {
		url = apiBaseURL + "speech"
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse JSON response
	var ttsResp MistralTTSResponse
	if err := json.Unmarshal(respBody, &ttsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(ttsResp.AudioData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %v", err)
	}

	return audioData, nil
}

// Voices available in Mistral TTS
// Note: Actual voices depend on API key. These are example voice IDs.
// Use /api/tts/voices endpoint to get dynamic list from Mistral API.
var AvailableVoices = []string{
	"c69964a6-ab8b-4f8a-9465-ec0925096ec8", // Paul - Neutral (en_us)
	"530e2e20-58e2-45d8-b0a5-4594f4915944", // Paul - Sad (en_us)
	"1024d823-a11e-43ee-bf3d-d440dccc0577", // Paul - Happy (en_us)
	"e3596645-b1af-469e-b857-f18ddedc7652", // Oliver - Neutral (en_gb)
	"a3e41ea8-020b-44c0-8d8b-f6cc03524e31", // Jane - Sarcasm (en_gb)
}

// Models available in Mistral TTS
var AvailableModels = []string{
	"voxtral-mini-tts-2603",
}
