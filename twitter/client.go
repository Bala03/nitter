package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bearerToken = "AAAAAAAAAAAAAAAAAAAAAFQODgEAAAAAVHTp76lzh3rFzcHbmHVvQxYYpTw%3DckAlMINMjmCwxUcaXbAN4XqJVdgMJaHqNOFgPMK0zN1qLqLQCF"
	apiBase     = "https://api.twitter.com"
	activateURL = apiBase + "/1.1/guest/activate.json"

	graphqlBase                  = apiBase + "/graphql"
	graphUserURL                 = graphqlBase + "/u7wQyGi6oExe8_TRWGMq4Q/UserResultByScreenNameQuery"
	graphUserByIdURL             = graphqlBase + "/oPppcargziU1uDQHAUmH-A/UserResultByIdQuery"
	graphUserTweetsURL           = graphqlBase + "/3JNH4e9dq1BifLxAa3UMWg/UserWithProfileTweetsQueryV2"
	graphUserTweetsAndRepliesURL = graphqlBase + "/8IS8MaO-2EN6GZZZb8jF0g/UserWithProfileTweetsAndRepliesQueryV2"
	graphUserMediaURL            = graphqlBase + "/PDfFf8hGeJvUCiTyWtw4wQ/MediaTimelineV2"
	graphTweetURL                = graphqlBase + "/83h5UyHZ9wEKBVzALX8R_g/ConversationTimelineV2"
	graphTweetResultURL          = graphqlBase + "/sITyJdhRPpvpEjg4waUmTA/TweetResultByIdQuery"
	graphSearchTimelineURL       = graphqlBase + "/gkjsKepM6gl_HmFWoWKfgg/SearchTimeline"
	graphListByIdURL             = graphqlBase + "/iTpgCtbdxrsJfyx0cFjHqg/ListByRestId"
	graphListBySlugURL           = graphqlBase + "/-kmqNvm5Y-cVrfvBy6docg/ListBySlug"
	graphListMembersURL          = graphqlBase + "/P4NpVZDqUD_7MEM84L-8nw/ListMembers"
	graphListTweetsURL           = graphqlBase + "/BbGLL1ZfMibdFNWlk7a0Pw/ListTimeline"

	maxConcurrentReqs = 5
	maxLastUse        = 1 * time.Hour
	maxAge            = 2*time.Hour + 55*time.Minute
	failDelay         = 30 * time.Minute
)

var gqlFeatures = strings.Join([]string{
	`"android_graphql_skip_api_media_color_palette":false`,
	`"blue_business_profile_image_shape_enabled":false`,
	`"creator_subscriptions_subscription_count_enabled":false`,
	`"creator_subscriptions_tweet_preview_api_enabled":true`,
	`"freedom_of_speech_not_reach_fetch_enabled":false`,
	`"graphql_is_translatable_rweb_tweet_is_translatable_enabled":false`,
	`"hidden_profile_likes_enabled":false`,
	`"highlights_tweets_tab_ui_enabled":false`,
	`"interactive_text_enabled":false`,
	`"longform_notetweets_consumption_enabled":true`,
	`"longform_notetweets_inline_media_enabled":false`,
	`"longform_notetweets_richtext_consumption_enabled":true`,
	`"longform_notetweets_rich_text_read_enabled":false`,
	`"responsive_web_edit_tweet_api_enabled":false`,
	`"responsive_web_enhance_cards_enabled":false`,
	`"responsive_web_graphql_exclude_directive_enabled":true`,
	`"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false`,
	`"responsive_web_graphql_timeline_navigation_enabled":false`,
	`"responsive_web_media_download_video_enabled":false`,
	`"responsive_web_text_conversations_enabled":false`,
	`"responsive_web_twitter_article_tweet_consumption_enabled":false`,
	`"responsive_web_twitter_blue_verified_badge_is_enabled":true`,
	`"rweb_lists_timeline_redesign_enabled":true`,
	`"spaces_2022_h2_clipping":true`,
	`"spaces_2022_h2_spaces_communities":true`,
	`"standardized_nudges_misinfo":false`,
	`"subscriptions_verification_info_enabled":true`,
	`"subscriptions_verification_info_reason_enabled":true`,
	`"subscriptions_verification_info_verified_since_enabled":true`,
	`"super_follow_badge_privacy_enabled":false`,
	`"super_follow_exclusive_tweet_notifications_enabled":false`,
	`"super_follow_tweet_api_enabled":false`,
	`"super_follow_user_api_enabled":false`,
	`"tweet_awards_web_tipping_enabled":false`,
	`"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":false`,
	`"tweetypie_unmention_optimization_enabled":false`,
	`"unified_cards_ad_metadata_container_dynamic_card_content_query_enabled":false`,
	`"verified_phone_label_enabled":false`,
	`"vibe_api_enabled":false`,
	`"view_counts_everywhere_api_enabled":false`,
}, ",")

type token struct {
	tok     string
	init    time.Time
	lastUse time.Time
	pending int
}

type Client struct {
	httpClient   *http.Client
	tokens       []*token
	mu           sync.Mutex
	lastFailed   time.Time
	enableDebug  bool
	minTokens    int
	sessionStore *SessionStore
	sessionIdx   int
}

func NewClient(minTokens int, enableDebug bool) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		minTokens:   minTokens,
		enableDebug: enableDebug,
	}
}

// SetSessionStore attaches a session store to the client
func (c *Client) SetSessionStore(store *SessionStore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStore = store
}

// GetSessionStore returns the attached session store
func (c *Client) GetSessionStore() *SessionStore {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionStore
}

func (c *Client) log(msg string) {
	if c.enableDebug {
		fmt.Println("[tokens]", msg)
	}
}

func (c *Client) fetchGuestToken() (*token, error) {
	c.mu.Lock()
	if time.Since(c.lastFailed) < failDelay {
		c.mu.Unlock()
		return nil, fmt.Errorf("token fetching paused")
	}
	c.mu.Unlock()

	req, err := http.NewRequest("POST", activateURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.mu.Lock()
		c.lastFailed = time.Now()
		c.mu.Unlock()
		return nil, fmt.Errorf("fetching token failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.mu.Lock()
		c.lastFailed = time.Now()
		c.mu.Unlock()
		fmt.Printf("[tokens] fetching token failed: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Println("[tokens] fetching tokens paused, resuming in 30 minutes")
		return nil, fmt.Errorf("fetching token failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		GuestToken string `json:"guest_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	now := time.Now()
	return &token{tok: result.GuestToken, init: now, lastUse: now}, nil
}

func (t *token) expired() bool {
	now := time.Now()
	return t.init.Before(now.Add(-maxAge)) || t.lastUse.Before(now.Add(-maxLastUse))
}

func (c *Client) getToken() (*token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, t := range c.tokens {
		if !t.expired() && t.pending < maxConcurrentReqs {
			t.pending++
			return t, nil
		}
	}

	c.mu.Unlock()
	t, err := c.fetchGuestToken()
	c.mu.Lock()

	if err != nil {
		return nil, err
	}

	t.pending = 1
	c.tokens = append(c.tokens, t)
	c.log("added new token to pool")
	return t, nil
}

func (c *Client) releaseToken(t *token) {
	if t == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	t.pending--
	t.lastUse = time.Now()

	if t.expired() {
		for i, tok := range c.tokens {
			if tok == t {
				c.tokens = append(c.tokens[:i], c.tokens[i+1:]...)
				break
			}
		}
	}
}

func (c *Client) fetch(rawURL string) ([]byte, error) {
	// Try authenticated session first
	if data, err := c.fetchWithSession(rawURL); err == nil {
		return data, nil
	}

	// Fall back to guest token
	t, err := c.getToken()
	if err != nil {
		return nil, fmt.Errorf("rate limited: %v", err)
	}
	defer c.releaseToken(t)

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("x-guest-token", t.tok)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// fetchWithSession attempts to fetch using an authenticated session
func (c *Client) fetchWithSession(rawURL string) ([]byte, error) {
	c.mu.Lock()
	store := c.sessionStore
	c.mu.Unlock()

	if store == nil {
		return nil, fmt.Errorf("no session store")
	}

	sessions := store.GetActive()
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active sessions")
	}

	// Round-robin through sessions
	c.mu.Lock()
	idx := c.sessionIdx % len(sessions)
	c.sessionIdx++
	c.mu.Unlock()

	session := sessions[idx]

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+loginBearerToken)
	req.Header.Set("x-csrf-token", session.CT0)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: session.AuthToken})
	req.AddCookie(&http.Cookie{Name: "ct0", Value: session.CT0})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		store.MarkError(session.Username, fmt.Sprintf("HTTP %d - session may be expired", resp.StatusCode))
		return nil, fmt.Errorf("session expired for @%s", session.Username)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	store.UpdateLastUsed(session.Username)
	return io.ReadAll(resp.Body)
}

func (c *Client) InitTokenPool() {
	go func() {
		for {
			c.mu.Lock()
			active := 0
			for _, t := range c.tokens {
				if !t.expired() {
					active++
				}
			}
			c.mu.Unlock()

			if active < c.minTokens {
				for i := 0; i < min(4, c.minTokens-active); i++ {
					t, err := c.fetchGuestToken()
					if err != nil {
						break
					}
					c.mu.Lock()
					c.tokens = append(c.tokens, t)
					c.mu.Unlock()
					c.log("added new token to pool")
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
}

// GetUser fetches a user profile by screen name
func (c *Client) GetUser(username string) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("empty username")
	}

	variables := fmt.Sprintf(`{"screen_name":"%s"}`, username)
	params := url.Values{}
	params.Set("variables", variables)
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphUserURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphUser(data)
}

// GetUserTweets fetches a user's tweets
func (c *Client) GetUserTweets(userID string, cursor string) (*Timeline, error) {
	if userID == "" {
		return nil, fmt.Errorf("empty user ID")
	}

	cursorPart := ""
	if cursor != "" {
		cursorPart = fmt.Sprintf(`"cursor":"%s",`, cursor)
	}
	variables := fmt.Sprintf(`{"rest_id":"%s",%s"count":20}`, userID, cursorPart)
	params := url.Values{}
	params.Set("variables", variables)
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphUserTweetsURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphTimeline(data, "user")
}

// GetTweet fetches a single tweet/conversation
func (c *Client) GetTweet(id string) (*Conversation, error) {
	if id == "" {
		return nil, fmt.Errorf("empty tweet ID")
	}

	variables := fmt.Sprintf(`{"focalTweetId":"%s","includeHasBirdwatchNotes":false}`, id)
	params := url.Values{}
	params.Set("variables", variables)
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphTweetURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphConversation(data, id)
}

// GetSearch performs a tweet search
func (c *Client) GetSearch(query string, cursor string) (*Timeline, error) {
	if query == "" {
		return &Timeline{Beginning: true}, nil
	}

	variables := map[string]interface{}{
		"rawQuery":                query,
		"count":                   20,
		"product":                 "Latest",
		"withDownvotePerspective": false,
		"withReactionsMetadata":   false,
		"withReactionsPerspective": false,
	}
	if cursor != "" {
		variables["cursor"] = cursor
	}

	varsJSON, _ := json.Marshal(variables)
	params := url.Values{}
	params.Set("variables", string(varsJSON))
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphSearchTimelineURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphSearchTimeline(data)
}

// GetUserMedia fetches a user's media tweets
func (c *Client) GetUserMedia(userID string, cursor string) (*Timeline, error) {
	if userID == "" {
		return nil, fmt.Errorf("empty user ID")
	}

	cursorPart := ""
	if cursor != "" {
		cursorPart = fmt.Sprintf(`"cursor":"%s",`, cursor)
	}
	variables := fmt.Sprintf(`{"rest_id":"%s",%s"count":20}`, userID, cursorPart)
	params := url.Values{}
	params.Set("variables", variables)
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphUserMediaURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphTimeline(data, "user")
}

// GetUserReplies fetches a user's replies
func (c *Client) GetUserReplies(userID string, cursor string) (*Timeline, error) {
	if userID == "" {
		return nil, fmt.Errorf("empty user ID")
	}

	cursorPart := ""
	if cursor != "" {
		cursorPart = fmt.Sprintf(`"cursor":"%s",`, cursor)
	}
	variables := fmt.Sprintf(`{"rest_id":"%s",%s"count":20}`, userID, cursorPart)
	params := url.Values{}
	params.Set("variables", variables)
	params.Set("features", "{"+gqlFeatures+"}")

	rawURL := graphUserTweetsAndRepliesURL + "?" + params.Encode()
	data, err := c.fetch(rawURL)
	if err != nil {
		return nil, err
	}

	return parseGraphTimeline(data, "user")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return strconv.Itoa(n)
}
