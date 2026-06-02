package parser

import (
	"encoding/base64"
	"fmt"
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
		{"[x-post]", true},
		{"[discussion]", true},
		{"[text]", true},
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

// =============================================================================
// Feed Type Detection Tests
// =============================================================================

func TestIsRedditSelfPost(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"[self]", true},
		{"[self-post]", true},
		{"[discussion]", true},
		{"[text]", true},
		{"This is not a self post", false},
		{"[Self]", true}, // Case insensitive
		{"[SELF]", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isRedditSelfPost(tt.input)
			if result != tt.expected {
				t.Errorf("isRedditSelfPost(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// WebSub Hub Detection Tests
// =============================================================================

func TestFindHubInXML(t *testing.T) {
	tests := []struct {
		name     string
		xmlStr   string
		expected string
	}{
		{
			name:     "Standard hub link",
			xmlStr:   `<rss><channel><link rel="hub" href="https://hub.example.com"/></channel></rss>`,
			expected: "https://hub.example.com",
		},
		{
			name:     "Hub link with single quotes",
			xmlStr:   `<rss><channel><link rel='hub' href='https://hub2.example.com'/></channel></rss>`,
			expected: "https://hub2.example.com",
		},
		{
			name:     "Atom hub link",
			xmlStr:   `<feed><link rel="hub" href="https://atom-hub.example.com"/></feed>`,
			expected: "https://atom-hub.example.com",
		},
		{
			name:     "No hub link",
			xmlStr:   `<rss><channel><title>Test</title></channel></rss>`,
			expected: "",
		},
		{
			name:     "Hub link in atom:link",
			xmlStr:   `<feed><atom:link rel="hub" href="https://atom-hub2.example.com"/></feed>`,
			expected: "https://atom-hub2.example.com",
		},
		{
			name:     "Empty XML",
			xmlStr:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findHubInXML(tt.xmlStr)
			if result != tt.expected {
				t.Errorf("findHubInXML(%q) = %q, expected %q", tt.xmlStr, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Content Extraction Tests
// =============================================================================

func TestExtractDescriptionAndContent(t *testing.T) {
	// We can't easily test extractDescriptionAndContent without creating a full gofeed.Item
	// So we test the helper functions that are accessible

	// Test Reddit content extraction with class="md"
	redditHTML := `<div class="md">This is the post content</div>`
	result := extractRedditContent(redditHTML)
	if result != "This is the post content" {
		t.Errorf("extractRedditContent failed: got %q", result)
	}

	// Test Reddit content extraction with fallback
	redditHTMLNoMD := `<div>Just some <b>content</b> here</div>`
	result = extractRedditContent(redditHTMLNoMD)
	// stripHTMLTagsAndImages removes tags and normalizes whitespace
	expected := "Just some content here"
	if result != expected {
		t.Errorf("extractRedditContent (fallback) = %q, expected %q", result, expected)
	}

	// Test Reddit content extraction with image URLs (should be removed)
	redditHTMLWithImages := `<div>Content <img src="https://i.redd.it/test.jpg"> more</div>`
	result = extractRedditContent(redditHTMLWithImages)
	// Image URLs should be stripped
	if result != "Content more" {
		t.Errorf("extractRedditContent with images = %q, expected 'Content more'", result)
	}

	// Test empty/short content returns empty
	result = extractRedditContent(`<div>[link]</div>`)
	if result != "" {
		t.Errorf("extractRedditContent short content = %q, expected empty", result)
	}
}

// =============================================================================
// URL Extraction Tests
// =============================================================================

func TestExtractImageURL(t *testing.T) {
	// This would require creating a gofeed.Item and gofeed.Feed
	// For now, we test the pattern matching in extractImageFromDescription

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "img tag with src",
			text:     `<p><img src="https://example.com/image.jpg" alt="test"/></p>`,
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "img tag with single quotes",
			text:     `<img src='https://example.com/image.png'>`,
			expected: "https://example.com/image.png",
		},
		{
			name:     "Direct image URL",
			text:     `Check this: https://example.com/photo.jpg`,
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "Multiple images",
			text:     `<img src="https://example.com/first.jpg"><img src="https://example.com/second.png">`,
			expected: "https://example.com/first.jpg", // Should return first
		},
		{
			name:     "No image",
			text:     `Just text content`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractImageFromDescription(tt.text)
			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("extractImageFromDescription(%q) = %q, expected to contain %q", tt.text, result, tt.expected)
			}
			if tt.expected == "" && result != "" {
				t.Errorf("extractImageFromDescription(%q) = %q, expected empty", tt.text, result)
			}
		})
	}
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestGenerateSecret_DifferentLengths(t *testing.T) {
	lengths := []int{16, 24, 32, 64}

	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			secret, err := GenerateSecret(length)
			if err != nil {
				t.Fatalf("GenerateSecret(%d) failed: %v", length, err)
			}
			if len(secret) == 0 {
				t.Error("Generated secret is empty")
			}
			// Verify it's valid base64
			_, err = base64.URLEncoding.DecodeString(secret)
			if err != nil {
				t.Errorf("Generated secret is not valid base64: %v", err)
			}
		})
	}
}

func TestGenerateSecret_InvalidLength(t *testing.T) {
	// Test with invalid length (should still work, just different output)
	secret, err := GenerateSecret(0)
	if err != nil {
		t.Fatalf("GenerateSecret(0) should not fail: %v", err)
	}
	if secret != "" {
		t.Logf("GenerateSecret(0) returned: %q", secret)
	}
}

func TestIsCommonDomain_CaseInsensitive(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://LEMONDE.FR/rss", true},
		{"https://LeFigaro.FR/feed", true},
		{"https://REDDIT.COM/r/test", true},
		{"https://Example.COM/feed", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isCommonDomain(tt.url)
			if result != tt.expected {
				t.Errorf("isCommonDomain(%q) = %v, expected %v (case insensitive)", tt.url, result, tt.expected)
			}
		})
	}
}

func TestStripHTMLTags_ComplexHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Nested tags",
			input:    `<div><p><span>Text</span></p></div>`,
			expected: "Text",
		},
		{
			name:     "Multiple paragraphs without space",
			input:    `<p>First</p><p>Second</p>`,
			// After removing tags: "FirstSecond", strings.Fields keeps it as one word
			expected: "FirstSecond",
		},
		{
			name:     "HTML entities",
			input:    `&lt;p&gt;Test&lt;/p&gt;`,
			expected: "<p>Test</p>",
		},
		{
			name:     "Mixed content",
			input:    `Before <b>bold</b> and <i>italic</i> After`,
			expected: "Before bold and italic After",
		},
		{
			name:     "Line breaks without space",
			input:    `<p>Line 1</p><br><p>Line 2</p>`,
			// After removing tags: "Line 1Line 2", strings.Fields normalizes whitespace but consecutive non-space chars stay together
			expected: "Line 1Line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTMLTags(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
