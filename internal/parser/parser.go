package parser

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"rss-aggregator/internal/logger"
	"rss-aggregator/internal/models"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// DefaultWebSubHub is the public WebSub hub that many publishers use
const DefaultWebSubHub = "https://pubsubhubbub.appspot.com"

// ParseFeed parses a feed URL and returns Feed and Items
func ParseFeed(url string) (*models.Feed, []models.Item, error) {
	logger.Debugf("Starting to parse feed: %s", url)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		logger.Errorf("Error parsing feed %s: %v", url, err)
		return nil, nil, err
	}
	logger.Debugf("Successfully parsed feed: %s (Title: %s)", url, feed.Title)

	// Extract WebSub information from feed
	feedURL := feed.Link
	if feed.FeedLink != "" {
		feedURL = feed.FeedLink
	}

	// Try to find hub from XML
	hubURL := findHubInXMLFromURL(feedURL)
	if hubURL == "" {
		// Use default hub for common domains
		if isCommonDomain(feedURL) {
			hubURL = DefaultWebSubHub
		}
	}

	// Generate a secret for WebSub verification
	secret, _ := GenerateSecret(32)

	resultFeed := &models.Feed{
		URL:       url,
		Title:     feed.Title,
		Link:      feed.Link,
		Type:      feed.FeedType,
		LastFetch: time.Now(),
		HubURL:    hubURL,
		TopicURL:  feedURL,
		Secret:    secret,
	}

	var items []models.Item
	for _, item := range feed.Items {
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}

		// Extract image URL from various sources
		imageURL := extractImageURL(item, feedURL)

		// Extract description, use content preview if description is empty or too short
		description, content := extractDescriptionAndContent(item)

		items = append(items, models.Item{
			Title:       item.Title,
			Link:        item.Link,
			Description: description,
			Content:     content,
			ImageURL:    imageURL,
			PublishedAt: published,
			Guid:        item.GUID,
		})
	}

	return resultFeed, items, nil
}

// isCommonDomain checks if the URL is from a domain that typically supports WebSub
func isCommonDomain(feedURL string) bool {
	commonDomains := []string{
		"lemonde.fr",
		"lefigaro.fr",
		"clubic.com",
		"reddit.com",
		"bbc.com",
		"nytimes.com",
		"washingtonpost.com",
		"theguardian.com",
		"wordpress.com",
		"blogspot.com",
		"medium.com",
		"dev.to",
	}

	for _, domain := range commonDomains {
		if strings.Contains(strings.ToLower(feedURL), domain) {
			return true
		}
	}
	return false
}

// findHubInXMLFromURL fetches a feed and looks for hub links in the XML
func findHubInXMLFromURL(feedURL string) string {
	resp, err := http.Get(feedURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10240))
	return findHubInXML(string(body))
}

// findHubInXML looks for hub links in raw XML
func findHubInXML(xmlStr string) string {
	lowerXML := strings.ToLower(xmlStr)

	// Look for <link rel="hub" href="..."> or <link rel='hub' href='...'>
	if strings.Contains(lowerXML, `<link rel="hub"`) || strings.Contains(lowerXML, `<link rel='hub'`) {
		start := strings.Index(lowerXML, `href="`)
		if start != -1 {
			start += 6
			end := strings.Index(lowerXML[start:], `"`)
			if end != -1 {
				return xmlStr[start : start+end]
			}
		}
		start = strings.Index(lowerXML, `href='`)
		if start != -1 {
			start += 6
			end := strings.Index(lowerXML[start:], `'`)
			if end != -1 {
				return xmlStr[start : start+end]
			}
		}
	}

	// Look for <atom:link rel="hub"> or <atom:link rel='hub'>
	if (strings.Contains(lowerXML, `<atom:link`) && strings.Contains(lowerXML, `rel="hub"`)) ||
		(strings.Contains(lowerXML, `<atom:link`) && strings.Contains(lowerXML, `rel='hub'`)) {
		start := strings.Index(lowerXML, `href="`)
		if start != -1 {
			start += 6
			end := strings.Index(lowerXML[start:], `"`)
			if end != -1 {
				return xmlStr[start : start+end]
			}
		}
		start = strings.Index(lowerXML, `href='`)
		if start != -1 {
			start += 6
			end := strings.Index(lowerXML[start:], `'`)
			if end != -1 {
				return xmlStr[start : start+end]
			}
		}
	}

	return ""
}

// GenerateSecret generates a random secret for WebSub verification
func GenerateSecret(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetHubAndTopic extracts WebSub hub and topic from a feed URL
func GetHubAndTopic(feedURL string) (string, string, error) {
	hubURL := findHubInXMLFromURL(feedURL)
	if hubURL == "" {
		if isCommonDomain(feedURL) {
			hubURL = DefaultWebSubHub
		} else {
			return "", feedURL, fmt.Errorf("no hub found for %s", feedURL)
		}
	}
	return hubURL, feedURL, nil
}

// extractImageURL extracts the best image URL from an item
func extractImageURL(item *gofeed.Item, feedURL string) string {
	// Check for direct image URL from Image field
	if item.Image != nil && item.Image.URL != "" {
		return item.Image.URL
	}

	// Check enclosures for images
	for _, enclosure := range item.Enclosures {
		// Prefer image types
		if strings.Contains(strings.ToLower(enclosure.Type), "image") {
			return enclosure.URL
		}
	}

	// For Reddit, check if description contains an image URL
	if strings.Contains(strings.ToLower(feedURL), "reddit.com") {
		// Try to extract image URL from description
		imgURL := extractImageFromDescription(item.Description)
		if imgURL != "" {
			return imgURL
		}
		// Try to extract from content
		imgURL = extractImageFromDescription(item.Content)
		if imgURL != "" {
			return imgURL
		}
	}

	// Fallback to first enclosure regardless of type
	if len(item.Enclosures) > 0 {
		return item.Enclosures[0].URL
	}

	return ""
}

// extractImageFromDescription extracts an image URL from HTML text
func extractImageFromDescription(text string) string {
	// Look for <img src="...">
	re := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	matches := re.FindStringSubmatch(strings.ToLower(text))
	if len(matches) > 1 {
		return html.UnescapeString(matches[1])
	}

	// Look for URLs ending with image extensions
	re = regexp.MustCompile(`(https?://[^\s""]+\.(?:jpg|jpeg|png|gif|webp|svg))`)
	matches = re.FindStringSubmatch(strings.ToLower(text))
	if len(matches) > 1 {
		return html.UnescapeString(matches[1])
	}

	return ""
}

// extractDescriptionAndContent extracts and improves description and content
func extractDescriptionAndContent(item *gofeed.Item) (string, string) {
	content := item.Content
	originalDesc := item.Description

	// For Reddit, extract content from HTML
	if strings.Contains(strings.ToLower(item.Link), "reddit.com") {
		// Try to extract the actual post content from HTML
		textContent := extractRedditContent(content)
		if textContent != "" {
			preview := textContent
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return preview, content
		}
		return textContent, content
	}

	// If description is empty or just a link, use content preview
	if originalDesc == "" || isJustALink(originalDesc) {
		// Strip HTML tags from content and take first part
		textContent := stripHTMLTags(content)
		if len(textContent) > 300 {
			textContent = textContent[:300] + "..."
		}
		return textContent, content
	}

	return originalDesc, content
}

// extractRedditContent extracts the post content from Reddit HTML
func extractRedditContent(htmlContent string) string {
	// Try to extract from <div class="md"> which contains self-post text
	re := regexp.MustCompile(`(?i)<div class="md">(.*?)</div>`)
	matches := re.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return stripHTMLTags(matches[1])
	}

	// Fallback: strip all HTML tags and clean up Reddit metadata
	text := stripHTMLTagsAndImages(htmlContent)

	// If after cleaning, text is empty or very short, return empty
	if len(text) < 10 {
		return ""
	}

	return text
}

// stripHTMLTagsAndImages removes HTML tags and image URLs from a string
func stripHTMLTagsAndImages(s string) string {
	// Remove all HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Remove image URLs (preview.redd.it, external-preview.redd.it, etc.)
	re = regexp.MustCompile(`https?://(external-)?preview\.redd\.it/[^\s]+`)
	s = re.ReplaceAllString(s, "")
	re = regexp.MustCompile(`https?://i\.redd\.it/[^\s]+`)
	s = re.ReplaceAllString(s, "")

	// Unescape HTML entities
	s = html.UnescapeString(s)

	// Remove Reddit submission metadata
	s = regexp.MustCompile(`(?i)submitted by /u/\w+`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)\[link\]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)\[comments\]`).ReplaceAllString(s, "")

	// Normalize whitespace
	s = strings.Join(strings.Fields(s), " ")

	return strings.TrimSpace(s)
}

// isJustALink checks if a string is just a URL or link text
func isJustALink(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	// Check if it's a URL
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		// If it's just a URL without much text
		if len(s) < 100 && !strings.Contains(s, " ") {
			return true
		}
	}
	// Check for common link patterns
	// These are Reddit-specific markers for post types
	if s == "[link]" || s == "[self]" || s == "[self-post]" || s == "[x-post]" ||
		s == "[discussion]" || s == "[text]" {
		return true
	}
	return false
}

// isRedditSelfPost checks if the description indicates a Reddit self-post
func isRedditSelfPost(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "[self]" || s == "[self-post]" || s == "[discussion]" || s == "[text]"
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(s string) string {
	// Remove all HTML tags (including script and style)
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Unescape HTML entities
	s = html.UnescapeString(s)

	// Normalize whitespace
	s = strings.Join(strings.Fields(s), " ")

	return strings.TrimSpace(s)
}
