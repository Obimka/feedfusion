package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"rss-aggregator/internal/config"
	"rss-aggregator/internal/models"
	"rss-aggregator/internal/parser"
	"rss-aggregator/internal/storage"
	"rss-aggregator/internal/websub"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"urlquery": func(s string) string {
		return url.QueryEscape(s)
	},
	"isReddit": func(s string) bool {
		return strings.Contains(strings.ToLower(s), "reddit.com")
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"toJSON": func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	},
}).ParseGlob("web/templates/*.html"))

func RegisterRoutes(r *mux.Router, db *gorm.DB) {
	// Login page - public access
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// If already logged in, redirect to home
		if _, err := r.Cookie("token"); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		tmpl.ExecuteTemplate(w, "login.html", nil)
	})

	// Logout page - public access
	r.HandleFunc("/logged-out", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "logout.html", nil)
	})

	// Settings page - requires authentication
	r.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated
		if _, err := r.Cookie("token"); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		tmpl.ExecuteTemplate(w, "settings.html", nil)
	})

	// Protected routes
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated
		if _, err := r.Cookie("token"); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit == 0 {
			limit = 20
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		// Get filter mode from query param
		filterMode := r.URL.Query().Get("filter")
		if filterMode == "" {
			filterMode = "included"
		}

		// Get view mode from query param
		viewMode := r.URL.Query().Get("view")
		if viewMode == "" {
			viewMode = "full"
		}

		// Get show read status from query param (default: false = hide read articles)
		showRead := r.URL.Query().Get("showRead") == "true"

		// Get feed IDs based on filter mode
		var feedIDs []uint
		if strings.HasPrefix(filterMode, "feed:") {
			// Filter by specific feed ID
			feedIDStr := strings.TrimPrefix(filterMode, "feed:")
			feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
			if err == nil {
				feedIDs = []uint{uint(feedID)}
			}
		} else if filterMode == "all" {
			// Get all feed IDs
			allFeeds, _ := storage.GetAllFeeds(db)
			for _, f := range allFeeds {
				feedIDs = append(feedIDs, f.ID)
			}
		} else {
			// Get only included feeds (default)
			includedFeeds, _ := storage.GetIncludedFeeds(db)
			for _, f := range includedFeeds {
				feedIDs = append(feedIDs, f.ID)
			}
		}

		items, count, _ := storage.GetFilteredItemsByRead(db, limit, offset, showRead, feedIDs)

		// Load feed titles for display
		feedMap := make(map[uint]string)
		allFeeds, _ := storage.GetAllFeeds(db)
		// Sort feeds by last fetch date (most recent first)
		for i := 0; i < len(allFeeds)-1; i++ {
			for j := i + 1; j < len(allFeeds); j++ {
				if allFeeds[j].LastFetch.After(allFeeds[i].LastFetch) {
					allFeeds[i], allFeeds[j] = allFeeds[j], allFeeds[i]
				}
			}
		}
		for _, f := range allFeeds {
			feedMap[f.ID] = f.Title
		}
		for i := range items {
			items[i].FeedTitle = feedMap[items[i].FeedID]
		}

		// Get included feed IDs for checkboxes
		includedFeeds, _ := storage.GetIncludedFeeds(db)
		var includedIDs []uint
		for _, f := range includedFeeds {
			includedIDs = append(includedIDs, f.ID)
		}

		nextOffset := offset + limit
		hasOlder := count > int64(nextOffset)
		hasNewer := offset > 0
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}

		// Load weather cities from config and convert to map for template
		cfg, _ := config.LoadConfig("data/config.yaml")
		weatherCities := cfg.WeatherCities
		if len(weatherCities) == 0 {
			// Default cities if config is empty
			weatherCities = []config.WeatherCity{
				{Name: "Paris", Lat: 48.8566, Lon: 2.3522, Timezone: "Europe/Paris"},
				{Name: "Lyon", Lat: 45.7640, Lon: 4.8357, Timezone: "Europe/Paris"},
				{Name: "Marseille", Lat: 43.2965, Lon: 5.3698, Timezone: "Europe/Paris"},
				{Name: "Toulouse", Lat: 43.6047, Lon: 1.4442, Timezone: "Europe/Paris"},
				{Name: "Bordeaux", Lat: 44.8378, Lon: -0.5792, Timezone: "Europe/Paris"},
				{Name: "Lille", Lat: 50.6292, Lon: 3.0573, Timezone: "Europe/Paris"},
				{Name: "Nantes", Lat: 47.2184, Lon: -1.5536, Timezone: "Europe/Paris"},
				{Name: "Nice", Lat: 43.7102, Lon: 7.2620, Timezone: "Europe/Paris"},
				{Name: "Dijon", Lat: 47.3166, Lon: 5.0166, Timezone: "Europe/Paris"},
			}
		}

		// Convert to map for easier JavaScript access
		weatherCityMap := make(map[string]config.WeatherCity)
		for _, city := range weatherCities {
			weatherCityMap[city.Name] = city
		}

		tmpl.ExecuteTemplate(w, "index.html", struct {
			Items          []models.Item
			AllFeeds       []models.Feed
			IncludedIDs    []uint
			Count          int64
			Limit          int
			Offset         int
			HasOlder       bool
			HasNewer       bool
			NextOffset     int
			PrevOffset     int
			FilterMode     string
			ViewMode       string
			ShowRead       bool
			WeatherCities  []config.WeatherCity
			WeatherCityMap map[string]config.WeatherCity
		}{
			Items:          items,
			AllFeeds:       allFeeds,
			IncludedIDs:    includedIDs,
			Count:          count,
			Limit:          limit,
			Offset:         offset,
			HasOlder:       hasOlder,
			HasNewer:       hasNewer,
			NextOffset:     nextOffset,
			PrevOffset:     prevOffset,
			FilterMode:     filterMode,
			ViewMode:       viewMode,
			ShowRead:       showRead,
			WeatherCities:  weatherCities,
			WeatherCityMap: weatherCityMap,
		})
	})

	r.HandleFunc("/add-feed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			url := r.FormValue("url")
			if url != "" {
				feed, items, err := parser.ParseFeed(url)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				feed.Include = true
				storage.AddFeed(db, feed, items)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		tmpl.ExecuteTemplate(w, "add_feed.html", nil)
	})

	r.HandleFunc("/item/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		storage.MarkItemAsRead(db, uint(id))
		// Preserve query params and redirect back
		query := r.URL.Query()
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})

	r.HandleFunc("/item/{id}/read-and-redirect", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		storage.MarkItemAsRead(db, uint(id))

		// Get the target URL from query param
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Decode the URL (it was encoded when passed as param)
		decodedURL, err := url.QueryUnescape(targetURL)
		if err != nil {
			decodedURL = targetURL
		}

		http.Redirect(w, r, decodedURL, http.StatusSeeOther)
	})

	// Mark as unread
	r.HandleFunc("/item/{id}/unread", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		storage.MarkItemAsUnread(db, uint(id))
		// Preserve query params
		query := r.URL.Query()
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})

	r.HandleFunc("/feed/{id}/toggle", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		storage.ToggleFeedInclude(db, uint(id))
		// Redirect back to the referring page, or to home if no referer
		referer := r.Referer()
		if referer != "" {
			// Extract query params from referer if it's from our site
			if strings.Contains(referer, "/") && !strings.Contains(referer, "/feed/") {
				http.Redirect(w, r, referer, http.StatusSeeOther)
				return
			}
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	r.HandleFunc("/feed/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		storage.DeleteFeed(db, uint(id))
		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
	})

	r.HandleFunc("/feed/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		if r.Method == "POST" {
			feed, _ := storage.GetFeedByID(db, uint(id))
			feed.URL = r.FormValue("url")
			feed.Title = r.FormValue("title")
			feed.Include = r.FormValue("include") == "on"
			storage.UpdateFeed(db, feed)

			_, items, err := parser.ParseFeed(feed.URL)
			if err == nil {
				storage.AddFeed(db, feed, items)
			}

			http.Redirect(w, r, "/feeds", http.StatusSeeOther)
			return
		}

		feed, _ := storage.GetFeedByID(db, uint(id))
		tmpl.ExecuteTemplate(w, "edit_feed.html", struct {
			Feed *models.Feed
		}{
			Feed: feed,
		})
	})

	// Toggle show read
	r.HandleFunc("/toggle-read", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// The button already sends the toggled value, just use it
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})

	r.HandleFunc("/feeds", func(w http.ResponseWriter, r *http.Request) {
		feeds, _ := storage.GetAllFeeds(db)
		tmpl.ExecuteTemplate(w, "feeds.html", feeds)
	})

	// WebSub callback endpoint
	r.HandleFunc("/websub/callback", websub.HandleCallback(db)).Methods("GET", "POST")

	// WebSub subscription management
	r.HandleFunc("/websub/subscribe/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		feed, _ := storage.GetFeedByID(db, uint(id))
		if feed == nil {
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		// Get the callback base URL from the host
		callbackBaseURL := "http://" + r.Host
		if r.TLS != nil {
			callbackBaseURL = "https://" + r.Host
		}

		if err := websub.SubscribeFeedToWebSub(db, feed, callbackBaseURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
	})

	r.HandleFunc("/websub/unsubscribe/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		feed, _ := storage.GetFeedByID(db, uint(id))
		if feed == nil {
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		callbackBaseURL := "http://" + r.Host
		if r.TLS != nil {
			callbackBaseURL = "https://" + r.Host
		}

		if err := websub.UnsubscribeFeedFromWebSub(db, feed, callbackBaseURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
	})
}
