package rss

import (
	"strings"
	"testing"
	"time"

	"github.com/zedeus/nitter/twitter"
)

func TestGenerateUserFeed_WithPhotos(t *testing.T) {
	user := &twitter.User{
		ID:       "12345",
		Username: "testuser",
		Fullname: "Test User",
		Bio:      "Test bio",
		UserPic:  "https://pbs.twimg.com/profile_images/test.jpg",
	}

	tweets := []twitter.Tweet{
		{
			ID:   100,
			User: *user,
			Text: "Check out this photo!",
			Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Photos: []twitter.Photo{
				{URL: "https://pbs.twimg.com/media/photo1.jpg", Width: 1200, Height: 800},
				{URL: "https://pbs.twimg.com/media/photo2.jpg", Width: 1920, Height: 1080},
			},
			Stats: twitter.TweetStats{Replies: 5, Retweets: 10, Likes: 50},
		},
	}

	feed, err := GenerateUserFeed(user, tweets, "http://localhost:8080")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	// Check RSS header
	if !strings.Contains(feed, `<rss version="2.0"`) {
		t.Error("Missing RSS version header")
	}
	if !strings.Contains(feed, `xmlns:media="http://search.yahoo.com/mrss/"`) {
		t.Error("Missing media namespace")
	}

	// Check channel info
	if !strings.Contains(feed, "<title>Test User / @testuser</title>") {
		t.Error("Missing or incorrect channel title")
	}

	// Check media enclosures (direct download links)
	if !strings.Contains(feed, `url="https://pbs.twimg.com/media/photo1.jpg?name=orig"`) {
		t.Error("Missing direct download URL for photo1")
	}
	if !strings.Contains(feed, `url="https://pbs.twimg.com/media/photo2.jpg?name=orig"`) {
		t.Error("Missing direct download URL for photo2")
	}

	// Check media:content or media:group (multiple images should create a group)
	if !strings.Contains(feed, "media:group") {
		t.Error("Missing media:group for multiple images")
	}
	if !strings.Contains(feed, `medium="image"`) {
		t.Error("Missing medium=image attribute")
	}
	if !strings.Contains(feed, `width="1200"`) {
		t.Error("Missing width attribute")
	}
	if !strings.Contains(feed, `height="800"`) {
		t.Error("Missing height attribute")
	}

	// Check description contains download links
	if !strings.Contains(feed, "Direct Download") {
		t.Error("Missing direct download link in description")
	}
	if !strings.Contains(feed, "[Image]") {
		t.Error("Missing [Image] indicator in description")
	}
}

func TestGenerateUserFeed_WithVideo(t *testing.T) {
	user := &twitter.User{
		ID:       "12345",
		Username: "videouser",
		Fullname: "Video User",
	}

	tweets := []twitter.Tweet{
		{
			ID:   200,
			User: *user,
			Text: "Watch this video!",
			Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Video: &twitter.Video{
				DurationMs: 30000,
				URL:        "https://video.twimg.com/ext_tw_video/video.mp4",
				Thumb:      "https://pbs.twimg.com/ext_tw_video_thumb/thumb.jpg",
				Available:  true,
				Variants: []twitter.VideoVariant{
					{ContentType: twitter.VideoTypeMP4, URL: "https://video.twimg.com/ext_tw_video/720p.mp4", Bitrate: 2176000},
					{ContentType: twitter.VideoTypeMP4, URL: "https://video.twimg.com/ext_tw_video/480p.mp4", Bitrate: 832000},
					{ContentType: twitter.VideoTypeMP4, URL: "https://video.twimg.com/ext_tw_video/360p.mp4", Bitrate: 256000},
					{ContentType: twitter.VideoTypeM3U8, URL: "https://video.twimg.com/ext_tw_video/pl.m3u8", Bitrate: 0},
				},
			},
			Stats: twitter.TweetStats{Replies: 2, Retweets: 5, Likes: 20},
		},
	}

	feed, err := GenerateUserFeed(user, tweets, "http://localhost:8080")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	// Check video enclosure with best quality (highest bitrate MP4)
	if !strings.Contains(feed, "720p.mp4") {
		t.Error("Missing best quality video URL (720p)")
	}
	if !strings.Contains(feed, `type="video/mp4"`) {
		t.Error("Missing video/mp4 content type in enclosure")
	}

	// Check media:content
	if !strings.Contains(feed, `medium="video"`) {
		t.Error("Missing medium=video attribute")
	}

	// Check duration
	if !strings.Contains(feed, `duration="30"`) {
		t.Error("Missing duration attribute (30 seconds)")
	}

	// Check description contains all video quality options
	if !strings.Contains(feed, "All video qualities") {
		t.Error("Missing all video qualities section")
	}
	if !strings.Contains(feed, "2176 kbps") {
		t.Error("Missing 2176 kbps variant info")
	}
	if !strings.Contains(feed, "832 kbps") {
		t.Error("Missing 832 kbps variant info")
	}
	if !strings.Contains(feed, "256 kbps") {
		t.Error("Missing 256 kbps variant info")
	}

	// Check [Video] indicator
	if !strings.Contains(feed, "[Video]") {
		t.Error("Missing [Video] indicator")
	}
	if !strings.Contains(feed, "Direct Download (MP4)") {
		t.Error("Missing direct download text for video")
	}
}

func TestGenerateUserFeed_WithGif(t *testing.T) {
	user := &twitter.User{
		ID:       "12345",
		Username: "gifuser",
		Fullname: "GIF User",
	}

	tweets := []twitter.Tweet{
		{
			ID:   300,
			User: *user,
			Text: "Look at this GIF!",
			Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Gif: &twitter.Gif{
				URL:   "https://video.twimg.com/tweet_video/gif.mp4",
				Thumb: "https://pbs.twimg.com/tweet_video_thumb/gif_thumb.jpg",
			},
			Stats: twitter.TweetStats{Replies: 1, Retweets: 3, Likes: 15},
		},
	}

	feed, err := GenerateUserFeed(user, tweets, "http://localhost:8080")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	// Check GIF enclosure
	if !strings.Contains(feed, "gif.mp4") {
		t.Error("Missing GIF URL")
	}

	// Check [GIF] indicator
	if !strings.Contains(feed, "[GIF]") {
		t.Error("Missing [GIF] indicator in description")
	}
	if !strings.Contains(feed, "Direct Download (MP4)") {
		t.Error("Missing direct download text for GIF")
	}

	// Check media:content medium
	if !strings.Contains(feed, `medium="video"`) {
		t.Error("Missing medium=video for GIF (GIFs are MP4)")
	}
}

func TestGenerateUserFeed_WithQuote(t *testing.T) {
	user := &twitter.User{
		ID:       "12345",
		Username: "quoteuser",
		Fullname: "Quote User",
	}

	quotedUser := twitter.User{
		Username: "original",
		Fullname: "Original Poster",
	}

	tweets := []twitter.Tweet{
		{
			ID:   400,
			User: *user,
			Text: "Great tweet!",
			Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Quote: &twitter.Tweet{
				ID:   50,
				User: quotedUser,
				Text: "This is the original tweet with a photo",
				Photos: []twitter.Photo{
					{URL: "https://pbs.twimg.com/media/quoted_photo.jpg", Width: 800, Height: 600},
				},
			},
			Stats: twitter.TweetStats{Replies: 0, Retweets: 1, Likes: 5},
		},
	}

	feed, err := GenerateUserFeed(user, tweets, "http://localhost:8080")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	// Check quoted tweet content
	if !strings.Contains(feed, "@original") {
		t.Error("Missing quoted tweet username")
	}
	if !strings.Contains(feed, "This is the original tweet") {
		t.Error("Missing quoted tweet text")
	}
	if !strings.Contains(feed, "[Quoted Image]") {
		t.Error("Missing quoted image indicator")
	}
	if !strings.Contains(feed, "quoted_photo.jpg") {
		t.Error("Missing quoted photo download link")
	}
}

func TestGenerateSearchFeed(t *testing.T) {
	tweets := []twitter.Tweet{
		{
			ID: 500,
			User: twitter.User{
				Username: "searchresult",
				Fullname: "Search Result",
			},
			Text: "This matches the search query",
			Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Photos: []twitter.Photo{
				{URL: "https://pbs.twimg.com/media/search_img.jpg", Width: 640, Height: 480},
			},
			Stats: twitter.TweetStats{Replies: 0, Retweets: 0, Likes: 1},
		},
	}

	feed, err := GenerateSearchFeed("test query", tweets, "http://localhost:8080")
	if err != nil {
		t.Fatalf("GenerateSearchFeed failed: %v", err)
	}

	if !strings.Contains(feed, "Search: test query") {
		t.Error("Missing search title")
	}
	if !strings.Contains(feed, "search_img.jpg") {
		t.Error("Missing search result media")
	}
}

func TestGetMediaAttachments(t *testing.T) {
	tweet := &twitter.Tweet{
		Photos: []twitter.Photo{
			{URL: "https://pbs.twimg.com/media/img1.jpg", Width: 1000, Height: 500},
		},
		Video: &twitter.Video{
			Available:  true,
			DurationMs: 15000,
			Thumb:      "https://pbs.twimg.com/thumb.jpg",
			Variants: []twitter.VideoVariant{
				{ContentType: twitter.VideoTypeMP4, URL: "https://video.twimg.com/720p.mp4", Bitrate: 2000000},
				{ContentType: twitter.VideoTypeMP4, URL: "https://video.twimg.com/480p.mp4", Bitrate: 800000},
			},
		},
		Gif: &twitter.Gif{
			URL:   "https://video.twimg.com/gif.mp4",
			Thumb: "https://pbs.twimg.com/gif_thumb.jpg",
		},
	}

	attachments := tweet.GetMediaAttachments()

	if len(attachments) != 3 {
		t.Fatalf("Expected 3 attachments, got %d", len(attachments))
	}

	// Photo
	if attachments[0].Type != twitter.MediaTypePhoto {
		t.Error("First attachment should be photo")
	}
	if !strings.Contains(attachments[0].DirectURL, "name=orig") {
		t.Error("Photo direct URL should include name=orig for highest quality")
	}

	// Video - should pick highest bitrate
	if attachments[1].Type != twitter.MediaTypeVideo {
		t.Error("Second attachment should be video")
	}
	if !strings.Contains(attachments[1].DirectURL, "720p.mp4") {
		t.Errorf("Video should pick highest bitrate variant, got %s", attachments[1].DirectURL)
	}
	if attachments[1].Duration != 15000 {
		t.Errorf("Video duration should be 15000, got %d", attachments[1].Duration)
	}

	// GIF
	if attachments[2].Type != twitter.MediaTypeGif {
		t.Error("Third attachment should be gif")
	}
	if attachments[2].DirectURL != "https://video.twimg.com/gif.mp4" {
		t.Error("GIF direct URL should be the video URL")
	}
}
