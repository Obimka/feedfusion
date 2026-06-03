package websub

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"rss-aggregator/internal/logger"
	"rss-aggregator/internal/models"
	"rss-aggregator/internal/parser"
	"rss-aggregator/internal/storage"

	"gorm.io/gorm"
)

// WebSubNotification represents a WebSub push notification
type WebSubNotification struct {
	Hub  string `json:"hub"`
	Self string `json:"self"`
}

// Subscribe subscribes to a WebSub hub for feed updates
func Subscribe(hubURL, topicURL, callbackURL, secret string) error {
	// Create the subscription request
	subscription := map[string]string{
		"hub.callback": callbackURL,
		"hub.mode":     "subscribe",
		"hub.topic":    topicURL,
		"hub.secret":   secret,
		"hub.verify":   "async",
	}

	// Convert to URL query parameters
	params := url.Values{}
	for key, value := range subscription {
		params.Set(key, value)
	}

	// Send POST request to hub
	resp, err := http.PostForm(hubURL, params)
	if err != nil {
		return fmt.Errorf("failed to subscribe to hub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub returned status %d: %s", resp.StatusCode, string(body))
	}

	// Verify the subscription by checking for the verification request
	// The hub should send a GET request to our callback with hub.challenge
	// We need to respond with the challenge value

	return nil
}

// Unsubscribe unsubscribes from a WebSub hub
func Unsubscribe(hubURL, topicURL, callbackURL, secret string) error {
	subscription := map[string]string{
		"hub.callback": callbackURL,
		"hub.mode":     "unsubscribe",
		"hub.topic":    topicURL,
		"hub.secret":   secret,
	}

	params := url.Values{}
	for key, value := range subscription {
		params.Set(key, value)
	}

	resp, err := http.PostForm(hubURL, params)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from hub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// VerifySignature verifies the WebSub signature
func VerifySignature(body []byte, secret, signature string) bool {
	// Decode the base64 signature
	decodedSig, err := base64.URLEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Create HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	// Compare signatures
	return hmac.Equal(decodedSig, expectedMAC)
}

// HandleCallback handles incoming WebSub notifications
func HandleCallback(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != http.MethodPost {
			// Handle GET for verification
			if r.Method == http.MethodGet {
				challenge := r.URL.Query().Get("hub.challenge")
				if challenge != "" {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte(challenge))
					return
				}
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Get the signature
		signature := r.Header.Get("X-Hub-Signature")
		if signature == "" {
			signature = r.Header.Get("X-Hub-Signature-Sha256")
		}

		// Get the feed by topic
		topic := r.Header.Get("X-Hub-Topic")
		if topic == "" {
			// Try to parse the notification
			var notification WebSubNotification
			if err := json.Unmarshal(body, &notification); err == nil {
				topic = notification.Self
			}
		}

		// Find the feed with this topic URL
		var feed models.Feed
		if err := db.Where("topic_url = ? OR url = ?", topic, topic).First(&feed).Error; err != nil {
			logger.Warnf("WebSub: No feed found for topic %s", topic)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Verify the signature
		if signature != "" && feed.Secret != "" {
			if !VerifySignature(body, feed.Secret, strings.TrimPrefix(signature, "sha256=")) {
				logger.Warnf("WebSub: Invalid signature for feed %s", feed.Title)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Parse the updated feed
		_, items, err := parser.ParseFeed(feed.URL)
		if err != nil {
			logger.Errorf("WebSub: Error parsing feed %s: %v", feed.URL, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Update the feed and add new items
		feed.LastFetch = time.Now()
		feed.Error = ""
		if err := db.Save(&feed).Error; err != nil {
			logger.Errorf("WebSub: Error saving feed %s: %v", feed.Title, err)
		}

		storage.AddFeed(db, &feed, items)
		logger.Infof("WebSub: Received update for %s, added %d items", feed.Title, len(items))

		// Return 200 OK
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// SubscribeFeedToWebSub subscribes a feed to its WebSub hub
func SubscribeFeedToWebSub(db *gorm.DB, feed *models.Feed, callbackBaseURL string) error {
	if feed.HubURL == "" {
		// Try to discover hub
		hubURL, topicURL, err := parser.GetHubAndTopic(feed.URL)
		if err != nil {
			logger.Warnf("WebSub: Could not discover hub for %s: %v", feed.URL, err)
			return err
		}
		feed.HubURL = hubURL
		feed.TopicURL = topicURL
		if err := db.Save(feed).Error; err != nil {
			return err
		}
	}

	if feed.HubURL == "" {
		logger.Warnf("WebSub: No hub found for feed %s", feed.URL)
		return nil
	}

	// Generate a secret if not set
	if feed.Secret == "" {
		secret, _ := parser.GenerateSecret(32)
		feed.Secret = secret
		if err := db.Save(feed).Error; err != nil {
			return err
		}
	}

	// Build callback URL
	callbackURL := fmt.Sprintf("%s/websub/callback?feed_id=%d", callbackBaseURL, feed.ID)

	// Subscribe to the hub
	if err := Subscribe(feed.HubURL, feed.TopicURL, callbackURL, feed.Secret); err != nil {
		logger.Errorf("WebSub: Failed to subscribe feed %s: %v", feed.URL, err)
		return err
	}

	// Mark as subscribed
	feed.Subscribed = true
	return db.Save(feed).Error
}

// UnsubscribeFeedFromWebSub unsubscribes a feed from its WebSub hub
func UnsubscribeFeedFromWebSub(db *gorm.DB, feed *models.Feed, callbackBaseURL string) error {
	if !feed.Subscribed || feed.HubURL == "" {
		return nil
	}

	callbackURL := fmt.Sprintf("%s/websub/callback?feed_id=%d", callbackBaseURL, feed.ID)

	if err := Unsubscribe(feed.HubURL, feed.TopicURL, callbackURL, feed.Secret); err != nil {
		logger.Errorf("WebSub: Failed to unsubscribe feed %s: %v", feed.URL, err)
		return err
	}

	feed.Subscribed = false
	return db.Save(feed).Error
}

// StartWebSubManager starts the WebSub subscription manager
func StartWebSubManager(db *gorm.DB, callbackBaseURL string) {
	// Subscribe all active feeds with WebSub support
	var feeds []models.Feed
	db.Where("is_active = ? AND hub_url != ?", true, "").Find(&feeds)

	for _, feed := range feeds {
		if !feed.Subscribed {
			if err := SubscribeFeedToWebSub(db, &feed, callbackBaseURL); err != nil {
				logger.Warnf("WebSub: Could not subscribe feed %s: %v", feed.URL, err)
			}
		}
	}
}

// VerifyWebSubIntent handles the WebSub intent verification
func VerifyWebSubIntent(w http.ResponseWriter, r *http.Request) {
	// This handles the GET request for verification
	challenge := r.URL.Query().Get("hub.challenge")
	if challenge != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(challenge))
		return
	}

	// Handle other intents
	http.Error(w, "Invalid intent", http.StatusBadRequest)
}

// SendPing sends a ping to verify the subscription is active
func SendPing(hubURL, topicURL, callbackURL, secret string) error {
	ping := map[string]string{
		"hub.mode":  "ping",
		"hub.topic": topicURL,
	}

	params := url.Values{}
	for key, value := range ping {
		params.Set(key, value)
	}

	// Create signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(params.Encode()))
	signature := "sha256=" + base64.URLEncoding.EncodeToString(mac.Sum(nil))

	// Send POST request
	client := &http.Client{}
	req, err := http.NewRequest("POST", hubURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Hub-Signature", signature)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CheckHubAvailability checks if a hub is available and responsive
func CheckHubAvailability(hubURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(hubURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check if it's a hub by looking for the hub metadata
	body, _ := io.ReadAll(resp.Body)
	return bytes.Contains(body, []byte("hub")) || resp.StatusCode == http.StatusOK
}
