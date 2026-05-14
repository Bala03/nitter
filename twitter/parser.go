package twitter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseGraphUser(data []byte) (*User, error) {
	var resp struct {
		Data struct {
			User struct {
				Result json.RawMessage `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse user response: %v", err)
	}

	return parseUserResult(resp.Data.User.Result)
}

func parseUserResult(data json.RawMessage) (*User, error) {
	var result struct {
		TypeName string `json:"__typename"`
		RestID   string `json:"rest_id"`
		Legacy   struct {
			Name            string `json:"name"`
			ScreenName      string `json:"screen_name"`
			Description     string `json:"description"`
			Location        string `json:"location"`
			URL             string `json:"url"`
			ProfileImageURL string `json:"profile_image_url_https"`
			ProfileBanner   string `json:"profile_banner_url"`
			FriendsCount    int    `json:"friends_count"`
			FollowersCount  int    `json:"followers_count"`
			StatusesCount   int    `json:"statuses_count"`
			FavouritesCount int    `json:"favourites_count"`
			MediaCount      int    `json:"media_count"`
			Verified        bool   `json:"verified"`
			Protected       bool   `json:"protected"`
			CreatedAt       string `json:"created_at"`
		} `json:"legacy"`
		IsBlueVerified bool `json:"is_blue_verified"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse user result: %v", err)
	}

	if result.TypeName == "UserUnavailable" {
		return &User{Suspended: true}, nil
	}

	joinDate, _ := time.Parse(time.RubyDate, result.Legacy.CreatedAt)

	user := &User{
		ID:        result.RestID,
		Username:  result.Legacy.ScreenName,
		Fullname:  result.Legacy.Name,
		Bio:       result.Legacy.Description,
		Location:  result.Legacy.Location,
		Website:   result.Legacy.URL,
		UserPic:   strings.Replace(result.Legacy.ProfileImageURL, "_normal", "", 1),
		Banner:    result.Legacy.ProfileBanner,
		Following: result.Legacy.FriendsCount,
		Followers: result.Legacy.FollowersCount,
		Tweets:    result.Legacy.StatusesCount,
		Likes:     result.Legacy.FavouritesCount,
		Media:     result.Legacy.MediaCount,
		Verified:  result.Legacy.Verified || result.IsBlueVerified,
		Protected: result.Legacy.Protected,
		JoinDate:  joinDate,
	}

	return user, nil
}

func parseGraphTimeline(data []byte, kind string) (*Timeline, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse timeline: %v", err)
	}

	timeline := &Timeline{}
	instructions := findInstructions(resp, kind)

	for _, inst := range instructions {
		instMap, ok := inst.(map[string]interface{})
		if !ok {
			continue
		}
		instType, _ := instMap["type"].(string)

		switch instType {
		case "TimelineAddEntries":
			entries, _ := instMap["entries"].([]interface{})
			for _, entry := range entries {
				entryMap, ok := entry.(map[string]interface{})
				if !ok {
					continue
				}
				entryID, _ := entryMap["entryId"].(string)

				if strings.HasPrefix(entryID, "tweet-") || strings.HasPrefix(entryID, "profile-conversation-") {
					tweet := extractTweetFromEntry(entryMap)
					if tweet != nil {
						timeline.Content = append(timeline.Content, *tweet)
					}
				} else if strings.HasPrefix(entryID, "cursor-bottom-") {
					timeline.Bottom = extractCursorValue(entryMap)
				} else if strings.HasPrefix(entryID, "cursor-top-") {
					timeline.Top = extractCursorValue(entryMap)
				}
			}
		}
	}

	return timeline, nil
}

func parseGraphConversation(data []byte, tweetID string) (*Conversation, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse conversation: %v", err)
	}

	conv := &Conversation{}
	instructions := findInstructions(resp, "tweet")

	for _, inst := range instructions {
		instMap, ok := inst.(map[string]interface{})
		if !ok {
			continue
		}
		instType, _ := instMap["type"].(string)

		if instType == "TimelineAddEntries" {
			entries, _ := instMap["entries"].([]interface{})
			for _, entry := range entries {
				entryMap, ok := entry.(map[string]interface{})
				if !ok {
					continue
				}
				entryID, _ := entryMap["entryId"].(string)

				if strings.Contains(entryID, tweetID) {
					tweet := extractTweetFromEntry(entryMap)
					if tweet != nil {
						conv.Tweet = *tweet
					}
				}
			}
		}
	}

	return conv, nil
}

func parseGraphSearchTimeline(data []byte) (*Timeline, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse search: %v", err)
	}

	timeline := &Timeline{}

	dataMap, _ := resp["data"].(map[string]interface{})
	searchByRaw, _ := dataMap["search_by_raw_query"].(map[string]interface{})
	searchTimeline, _ := searchByRaw["search_timeline"].(map[string]interface{})
	tlMap, _ := searchTimeline["timeline"].(map[string]interface{})
	instructions, _ := tlMap["instructions"].([]interface{})

	for _, inst := range instructions {
		instMap, ok := inst.(map[string]interface{})
		if !ok {
			continue
		}
		instType, _ := instMap["type"].(string)

		if instType == "TimelineAddEntries" {
			entries, _ := instMap["entries"].([]interface{})
			for _, entry := range entries {
				entryMap, ok := entry.(map[string]interface{})
				if !ok {
					continue
				}
				entryID, _ := entryMap["entryId"].(string)

				if strings.HasPrefix(entryID, "tweet-") {
					tweet := extractTweetFromEntry(entryMap)
					if tweet != nil {
						timeline.Content = append(timeline.Content, *tweet)
					}
				} else if strings.HasPrefix(entryID, "cursor-bottom-") {
					timeline.Bottom = extractCursorValue(entryMap)
				}
			}
		}
	}

	return timeline, nil
}

func findInstructions(resp map[string]interface{}, kind string) []interface{} {
	dataMap, _ := resp["data"].(map[string]interface{})
	if dataMap == nil {
		return nil
	}

	switch kind {
	case "user":
		userMap, _ := dataMap["user"].(map[string]interface{})
		if userMap == nil {
			return nil
		}
		resultMap, _ := userMap["result"].(map[string]interface{})
		if resultMap == nil {
			return nil
		}
		timelineV2, _ := resultMap["timeline_v2"].(map[string]interface{})
		if timelineV2 == nil {
			return nil
		}
		timeline, _ := timelineV2["timeline"].(map[string]interface{})
		if timeline == nil {
			return nil
		}
		instructions, _ := timeline["instructions"].([]interface{})
		return instructions

	case "tweet":
		threaded, _ := dataMap["threaded_conversation_with_injections_v2"].(map[string]interface{})
		if threaded == nil {
			return nil
		}
		instructions, _ := threaded["instructions"].([]interface{})
		return instructions
	}

	return nil
}

func extractTweetFromEntry(entry map[string]interface{}) *Tweet {
	content, _ := entry["content"].(map[string]interface{})
	if content == nil {
		return nil
	}

	entryType, _ := content["entryType"].(string)
	if entryType == "TimelineTimelineItem" {
		itemContent, _ := content["itemContent"].(map[string]interface{})
		if itemContent == nil {
			return nil
		}
		return parseTweetResult(itemContent)
	} else if entryType == "TimelineTimelineModule" {
		items, _ := content["items"].([]interface{})
		if len(items) > 0 {
			item, _ := items[0].(map[string]interface{})
			if item != nil {
				itemObj, _ := item["item"].(map[string]interface{})
				if itemObj != nil {
					itemContent, _ := itemObj["itemContent"].(map[string]interface{})
					if itemContent != nil {
						return parseTweetResult(itemContent)
					}
				}
			}
		}
	}

	return nil
}

func parseTweetResult(itemContent map[string]interface{}) *Tweet {
	tweetResults, _ := itemContent["tweet_results"].(map[string]interface{})
	if tweetResults == nil {
		return nil
	}
	result, _ := tweetResults["result"].(map[string]interface{})
	if result == nil {
		return nil
	}

	typeName, _ := result["__typename"].(string)
	if typeName == "TweetWithVisibilityResults" {
		result, _ = result["tweet"].(map[string]interface{})
		if result == nil {
			return nil
		}
	}

	return parseTweetObject(result)
}

func parseTweetObject(result map[string]interface{}) *Tweet {
	legacy, _ := result["legacy"].(map[string]interface{})
	if legacy == nil {
		return nil
	}

	core, _ := result["core"].(map[string]interface{})
	var user User
	if core != nil {
		userResults, _ := core["user_results"].(map[string]interface{})
		if userResults != nil {
			userResultData, _ := userResults["result"].(map[string]interface{})
			if userResultData != nil {
				userJSON, _ := json.Marshal(userResultData)
				parsed, err := parseUserResult(userJSON)
				if err == nil && parsed != nil {
					user = *parsed
				}
			}
		}
	}

	idStr, _ := legacy["id_str"].(string)
	id, _ := strconv.ParseInt(idStr, 10, 64)
	fullText, _ := legacy["full_text"].(string)
	createdAt, _ := legacy["created_at"].(string)

	tweetTime, _ := time.Parse(time.RubyDate, createdAt)

	tweet := &Tweet{
		ID:        id,
		User:      user,
		Text:      fullText,
		Time:      tweetTime,
		Available: true,
	}

	// Parse stats
	if replyCount, ok := legacy["reply_count"].(float64); ok {
		tweet.Stats.Replies = int(replyCount)
	}
	if rtCount, ok := legacy["retweet_count"].(float64); ok {
		tweet.Stats.Retweets = int(rtCount)
	}
	if likeCount, ok := legacy["favorite_count"].(float64); ok {
		tweet.Stats.Likes = int(likeCount)
	}
	if quoteCount, ok := legacy["quote_count"].(float64); ok {
		tweet.Stats.Quotes = int(quoteCount)
	}

	// Parse reply info
	if replyTo, ok := legacy["in_reply_to_screen_name"].(string); ok && replyTo != "" {
		tweet.Reply = append(tweet.Reply, replyTo)
	}
	if replyID, ok := legacy["in_reply_to_status_id_str"].(string); ok && replyID != "" {
		tweet.ReplyID, _ = strconv.ParseInt(replyID, 10, 64)
	}

	// Parse media (photos, videos, gifs)
	parseMediaEntities(legacy, tweet)

	// Parse retweet
	if rtResult, ok := result["legacy"].(map[string]interface{}); ok {
		if retweetedStatus, ok := rtResult["retweeted_status_result"].(map[string]interface{}); ok {
			rtResultObj, _ := retweetedStatus["result"].(map[string]interface{})
			if rtResultObj != nil {
				tweet.Retweet = parseTweetObject(rtResultObj)
			}
		}
	}

	// Parse quoted tweet
	if quotedResult, ok := result["quoted_status_result"].(map[string]interface{}); ok {
		quotedResultObj, _ := quotedResult["result"].(map[string]interface{})
		if quotedResultObj != nil {
			tweet.Quote = parseTweetObject(quotedResultObj)
		}
	}

	return tweet
}

func parseMediaEntities(legacy map[string]interface{}, tweet *Tweet) {
	extEntities, _ := legacy["extended_entities"].(map[string]interface{})
	if extEntities == nil {
		entities, _ := legacy["entities"].(map[string]interface{})
		if entities == nil {
			return
		}
		extEntities = entities
	}

	mediaList, _ := extEntities["media"].([]interface{})
	for _, m := range mediaList {
		media, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		mediaType, _ := media["type"].(string)
		mediaURL, _ := media["media_url_https"].(string)

		sizes, _ := media["sizes"].(map[string]interface{})
		width, height := 0, 0
		if sizes != nil {
			if large, ok := sizes["large"].(map[string]interface{}); ok {
				if w, ok := large["w"].(float64); ok {
					width = int(w)
				}
				if h, ok := large["h"].(float64); ok {
					height = int(h)
				}
			}
		}

		switch mediaType {
		case "photo":
			tweet.Photos = append(tweet.Photos, Photo{
				URL:    mediaURL,
				Width:  width,
				Height: height,
			})

		case "video":
			videoInfo, _ := media["video_info"].(map[string]interface{})
			if videoInfo == nil {
				continue
			}
			video := parseVideoInfo(videoInfo, mediaURL)
			tweet.Video = &video

		case "animated_gif":
			videoInfo, _ := media["video_info"].(map[string]interface{})
			if videoInfo == nil {
				continue
			}
			variants, _ := videoInfo["variants"].([]interface{})
			gifURL := ""
			for _, v := range variants {
				variant, ok := v.(map[string]interface{})
				if !ok {
					continue
				}
				varURL, _ := variant["url"].(string)
				if varURL != "" {
					gifURL = varURL
					break
				}
			}
			tweet.Gif = &Gif{
				URL:   gifURL,
				Thumb: mediaURL,
			}
		}
	}
}

func parseVideoInfo(videoInfo map[string]interface{}, thumbURL string) Video {
	video := Video{
		Thumb:     thumbURL,
		Available: true,
	}

	if duration, ok := videoInfo["duration_millis"].(float64); ok {
		video.DurationMs = int(duration)
	}

	variants, _ := videoInfo["variants"].([]interface{})
	for _, v := range variants {
		variant, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		varURL, _ := variant["url"].(string)
		contentType, _ := variant["content_type"].(string)
		bitrate, _ := variant["bitrate"].(float64)

		vv := VideoVariant{
			URL:     varURL,
			Bitrate: int(bitrate),
		}
		switch contentType {
		case "video/mp4":
			vv.ContentType = VideoTypeMP4
		case "application/x-mpegURL":
			vv.ContentType = VideoTypeM3U8
		default:
			vv.ContentType = VideoType(contentType)
		}
		video.Variants = append(video.Variants, vv)
	}

	// Set the best URL
	best := getBestVideoVariant(video.Variants)
	if best != nil {
		video.URL = best.URL
		video.PlaybackType = best.ContentType
	}

	return video
}

func extractCursorValue(entry map[string]interface{}) string {
	content, _ := entry["content"].(map[string]interface{})
	if content == nil {
		return ""
	}
	value, _ := content["value"].(string)
	if value != "" {
		return value
	}
	itemContent, _ := content["itemContent"].(map[string]interface{})
	if itemContent != nil {
		value, _ = itemContent["value"].(string)
	}
	return value
}
