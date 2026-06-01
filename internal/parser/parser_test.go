package parser

import (
	"strings"
	"testing"
)

func TestIsCommonDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://lemonde.fr/rss", true},
		{"https://lefigaro.fr/feed", true},
		{"https://reddit.com/r/golang/.rss", true},
		{"https://bbc.com/news/feed", true},
		{"https://example.com/feed", false},
		{"https://localhost:8080/feed", false},
		{"https://GITHUB.COM/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isCommonDomain(tt.url)
			if result != tt.expected {
				t.Errorf("isCommonDomain(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret(32)
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Error("Generated secret is empty")
	}
	// Base64 encoded 32 bytes should be 44 characters (32*8/6 rounded up)
	if len(secret) != 44 {
		t.Errorf("Expected secret length 44, got: %d", len(secret))
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<div><span>Test</span></div>", "Test"},
		{"Hello <b>World</b>!", "Hello World!"},
		{"<script>alert('xss')</script>Safe", "alert('xss')Safe"},
		{"No HTML here", "No HTML here"},
		{"<a href=\"#\">Link</a>", "Link"},
		{"  <p>  Spaced  </p>  ", "Spaced"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTMLTags(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractImageFromDescription(t *testing.T) {
	tests := []struct {
		input    string
		contains string // what the result should contain (not exact match)
	}{
		{
			input:    `<img src="https://example.com/image.jpg" alt="test">`,
			contains: "https://example.com/image.jpg",
		},
		{
			input:    `<img src='https://example.com/image.png'>`,
			contains: "https://example.com/image.png",
		},
		{
			input:    `Check out this image: https://example.com/photo.jpeg`,
			contains: "https://example.com/photo.jpeg",
		},
		{
			input:    `No image here`,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractImageFromDescription(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("extractImageFromDescription(%q) = %q, expected to contain %q", tt.input, result, tt.contains)
			}
			if tt.contains == "" && result != "" {
				t.Errorf("extractImageFromDescription(%q) = %q, expected empty", tt.input, result)
			}
		})
	}
}

func TestIsJustALink(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://test.com/path", true},
		{"[link]", true},
		{"[self]", true},
		{"[self-post]", true},
		{"This is a description with https://example.com", false},
		{"A normal description", false},
		{"", true},
		{"   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isJustALink(tt.input)
			if result != tt.expected {
				t.Errorf("isJustALink(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
