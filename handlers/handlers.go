package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zedeus/nitter/cache"
	"github.com/zedeus/nitter/config"
	"github.com/zedeus/nitter/rss"
	"github.com/zedeus/nitter/twitter"
)

type Handler struct {
	cfg       *config.Config
	client    *twitter.Client
	cache     *cache.RedisCache
	templates *template.Template
}

func New(cfg *config.Config, client *twitter.Client, redisCache *cache.RedisCache) (*Handler, error) {
	funcMap := template.FuncMap{
		"formatNumber": formatNumber,
		"formatDate":   formatDate,
		"tweetURL":     tweetURL,
		"profileURL":   profileURL,
		"mediaTypes":   mediaTypes,
		"hasMedia":     hasMedia,
		"safeHTML":     safeHTML,
		"truncate":     truncate,
		"timeAgo":      timeAgo,
		"add":          func(a, b int) int { return a + b },
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{})
			for i := 0; i < len(values)-1; i += 2 {
				key, _ := values[i].(string)
				dict[key] = values[i+1]
			}
			return dict
		},
		"string": func(v interface{}) string {
			return fmt.Sprintf("%v", v)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("parse templates: %v", err)
	}

	return &Handler{
		cfg:       cfg,
		client:    client,
		cache:     redisCache,
		templates: tmpl,
	}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Static files
	fs := http.FileServer(http.Dir(h.cfg.StaticDir))
	mux.Handle("/css/", fs)
	mux.Handle("/fonts/", fs)
	mux.Handle("/icons/", fs)
	mux.Handle("/favicon.ico", fs)
	mux.Handle("/favicon-32x32.png", fs)
	mux.Handle("/favicon-16x16.png", fs)
	mux.Handle("/apple-touch-icon.png", fs)
	mux.Handle("/logo.png", fs)
	mux.Handle("/site.webmanifest", fs)
	mux.Handle("/safari-pinned-tab.svg", fs)
	mux.Handle("/screenshot.png", fs)

	// Pages
	mux.HandleFunc("/", h.handleHome)
	mux.HandleFunc("/about", h.handleAbout)
	mux.HandleFunc("/settings", h.handleSettings)
	mux.HandleFunc("/search", h.handleSearch)
	mux.HandleFunc("/search/rss", h.handleSearchRSS)
}

func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/" {
		h.render(w, "home.html", map[string]interface{}{
			"cfg": h.cfg,
		})
		return
	}

	// Route: /@name or /name
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	name := strings.TrimPrefix(parts[0], "@")

	if name == "" {
		h.renderError(w, "Page not found", http.StatusNotFound)
		return
	}

	// Route: /name/status/id
	if len(parts) >= 3 && parts[1] == "status" {
		h.handleTweet(w, r, name, parts[2])
		return
	}

	// Route: /name/rss
	if len(parts) == 2 && parts[1] == "rss" {
		h.handleUserRSS(w, r, name)
		return
	}

	// Route: /name/media/rss or /name/with_replies/rss
	if len(parts) == 3 && parts[2] == "rss" {
		h.handleUserTabRSS(w, r, name, parts[1])
		return
	}

	// Route: /name/media or /name/with_replies
	if len(parts) == 2 && (parts[1] == "media" || parts[1] == "with_replies") {
		h.handleProfile(w, r, name, parts[1])
		return
	}

	// Default: profile page
	h.handleProfile(w, r, name, "tweets")
}

func (h *Handler) handleProfile(w http.ResponseWriter, r *http.Request, name string, tab string) {
	user, err := h.client.GetUser(name)
	if err != nil {
		h.renderError(w, fmt.Sprintf("User @%s not found: %v", name, err), http.StatusNotFound)
		return
	}

	if user.Suspended {
		h.renderError(w, fmt.Sprintf("User @%s is suspended", name), http.StatusForbidden)
		return
	}

	cursor := r.URL.Query().Get("cursor")
	var timeline *twitter.Timeline

	switch tab {
	case "media":
		timeline, err = h.client.GetUserMedia(user.ID, cursor)
	case "with_replies":
		timeline, err = h.client.GetUserReplies(user.ID, cursor)
	default:
		timeline, err = h.client.GetUserTweets(user.ID, cursor)
	}

	if err != nil {
		timeline = &twitter.Timeline{}
	}

	h.render(w, "profile.html", map[string]interface{}{
		"cfg":      h.cfg,
		"user":     user,
		"tweets":   timeline.Content,
		"tab":      tab,
		"cursor":   timeline.Bottom,
		"hasPrev":  timeline.Top != "",
		"prevCursor": timeline.Top,
	})
}

func (h *Handler) handleTweet(w http.ResponseWriter, r *http.Request, name string, idStr string) {
	conv, err := h.client.GetTweet(idStr)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Tweet not found: %v", err), http.StatusNotFound)
		return
	}

	h.render(w, "tweet.html", map[string]interface{}{
		"cfg":   h.cfg,
		"tweet": conv.Tweet,
		"conv":  conv,
	})
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.render(w, "search.html", map[string]interface{}{
			"cfg": h.cfg,
		})
		return
	}

	cursor := r.URL.Query().Get("cursor")
	timeline, err := h.client.GetSearch(query, cursor)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.render(w, "search.html", map[string]interface{}{
		"cfg":    h.cfg,
		"query":  query,
		"tweets": timeline.Content,
		"cursor": timeline.Bottom,
	})
}

func (h *Handler) handleAbout(w http.ResponseWriter, r *http.Request) {
	h.render(w, "about.html", map[string]interface{}{
		"cfg": h.cfg,
	})
}

func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		theme := r.FormValue("theme")
		if theme != "" {
			http.SetCookie(w, &http.Cookie{
				Name:   "theme",
				Value:  theme,
				Path:   "/",
				MaxAge: 365 * 24 * 3600,
			})
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	theme := "Nitter"
	if cookie, err := r.Cookie("theme"); err == nil {
		theme = cookie.Value
	}

	h.render(w, "settings.html", map[string]interface{}{
		"cfg":   h.cfg,
		"theme": theme,
	})
}

// RSS Handlers

func (h *Handler) handleUserRSS(w http.ResponseWriter, r *http.Request, name string) {
	if !h.cfg.EnableRSS {
		h.renderError(w, "RSS feeds are disabled", http.StatusForbidden)
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf("rss:user:%s", name)
	cached, _ := h.cache.Get(cacheKey)
	if cached != "" {
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Write([]byte(cached))
		return
	}

	user, err := h.client.GetUser(name)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	timeline, err := h.client.GetUserTweets(user.ID, "")
	if err != nil {
		http.Error(w, "Failed to fetch tweets", http.StatusInternalServerError)
		return
	}

	feed, err := rss.GenerateUserFeed(user, timeline.Content, h.cfg.BaseURL())
	if err != nil {
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
		return
	}

	// Cache the feed
	h.cache.Set(cacheKey, feed, time.Duration(h.cfg.RSSCacheTime)*time.Minute)

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(feed))
}

func (h *Handler) handleUserTabRSS(w http.ResponseWriter, r *http.Request, name string, tab string) {
	if !h.cfg.EnableRSS {
		h.renderError(w, "RSS feeds are disabled", http.StatusForbidden)
		return
	}

	if tab != "media" && tab != "with_replies" {
		http.Error(w, "Invalid tab", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("rss:%s:%s", tab, name)
	cached, _ := h.cache.Get(cacheKey)
	if cached != "" {
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Write([]byte(cached))
		return
	}

	user, err := h.client.GetUser(name)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var timeline *twitter.Timeline
	switch tab {
	case "media":
		timeline, err = h.client.GetUserMedia(user.ID, "")
	case "with_replies":
		timeline, err = h.client.GetUserReplies(user.ID, "")
	}
	if err != nil {
		http.Error(w, "Failed to fetch tweets", http.StatusInternalServerError)
		return
	}

	feed, err := rss.GenerateUserFeed(user, timeline.Content, h.cfg.BaseURL())
	if err != nil {
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
		return
	}

	h.cache.Set(cacheKey, feed, time.Duration(h.cfg.RSSCacheTime)*time.Minute)

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(feed))
}

func (h *Handler) handleSearchRSS(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.EnableRSS {
		h.renderError(w, "RSS feeds are disabled", http.StatusForbidden)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query", http.StatusBadRequest)
		return
	}

	timeline, err := h.client.GetSearch(query, "")
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	feed, err := rss.GenerateSearchFeed(query, timeline.Content, h.cfg.BaseURL())
	if err != nil {
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(feed))
}

// Template helpers

func (h *Handler) render(w http.ResponseWriter, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) renderError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	h.render(w, "error.html", map[string]interface{}{
		"cfg":     h.cfg,
		"message": message,
		"code":    code,
	})
}

// Template functions

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return strconv.Itoa(n)
}

func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func tweetURL(tweet twitter.Tweet) string {
	return fmt.Sprintf("/%s/status/%d", tweet.User.Username, tweet.ID)
}

func profileURL(user twitter.User) string {
	return fmt.Sprintf("/%s", user.Username)
}

func mediaTypes(tweet twitter.Tweet) []twitter.MediaType {
	return tweet.GetMediaTypes()
}

func hasMedia(tweet twitter.Tweet) bool {
	return len(tweet.Photos) > 0 || tweet.Video != nil || tweet.Gif != nil
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return t.Format("Jan 2")
	}
}
