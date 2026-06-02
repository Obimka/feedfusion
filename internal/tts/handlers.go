package tts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// MistralVoice represents a voice from Mistral TTS API
type MistralVoice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	Languages []string `json:"languages"`
	Gender    string   `json:"gender"`
	Age       int      `json:"age"`
	Tags      []string `json:"tags"`
	Color     string   `json:"color"`
}

// MistralVoicesResponse represents the response from Mistral voices API
type MistralVoicesResponse struct {
	Items      []MistralVoice `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// TTSHandler handles TTS requests
type TTSHandler struct {
	config TTSConfig
}

// NewTTSHandler creates a new TTSHandler
func NewTTSHandler(apiKey string) *TTSHandler {
	config := DefaultTTSConfig()
	config.APIKey = apiKey
	return &TTSHandler{config: config}
}

// GenerateSpeechHandler handles POST requests to generate speech from text
// Request body: {"text": "Hello world", "voice": "fr_FR", "model": "mistral-small"}
// Response: WAV audio data or error
func (h *TTSHandler) GenerateSpeechHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var request struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use defaults if not provided
	if request.Voice == "" {
		request.Voice = h.config.VoiceID
	}
	if request.Model == "" {
		request.Model = h.config.Model
	}

	// Generate speech
	audioData, err := GenerateSpeech(
		request.Text,
		request.Voice,
		request.Model,
		h.config.APIKey,
		h.config.APIBaseURL,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set headers for audio response
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "attachment; filename=speech.wav")

	// Write audio data
	w.Write(audioData)
}

// GetVoicesHandler returns the list of available voices from Mistral API
func (h *TTSHandler) GetVoicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to fetch voices from Mistral API
	voices, err := fetchMistralVoices(h.config.APIKey)
	if err != nil {
		// Fallback to static list
		json.NewEncoder(w).Encode(AvailableVoices)
		return
	}

	json.NewEncoder(w).Encode(voices)
}

// GetModelsHandler returns the list of available models
func (h *TTSHandler) GetModelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Return static list for now - Mistral TTS currently has one main model
	models := []string{"voxtral-mini-tts-2603"}
	json.NewEncoder(w).Encode(models)
}

// fetchMistralVoices fetches available voices from Mistral API
func fetchMistralVoices(apiKey string) ([]MistralVoice, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}

	url := "https://api.mistral.ai/v1/audio/voices?limit=100"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response MistralVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Items, nil
}

// StatusHandler returns whether TTS is enabled
func (h *TTSHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": h.config.APIKey != ""})
}

// RegisterRoutes registers TTS routes on the router
func (h *TTSHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/tts/speak", h.GenerateSpeechHandler).Methods("POST")
	router.HandleFunc("/api/tts/voices", h.GetVoicesHandler).Methods("GET")
	router.HandleFunc("/api/tts/models", h.GetModelsHandler).Methods("GET")
	router.HandleFunc("/api/tts/status", h.StatusHandler).Methods("GET")
}
