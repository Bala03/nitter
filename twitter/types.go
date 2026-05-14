package twitter

import "time"

type VideoType string

const (
	VideoTypeM3U8 VideoType = "application/x-mpegURL"
	VideoTypeMP4  VideoType = "video/mp4"
	VideoTypeVMAP VideoType = "video/vmap"
)

type VideoVariant struct {
	ContentType VideoType `json:"content_type"`
	URL         string    `json:"url"`
	Bitrate     int       `json:"bitrate"`
}

type Video struct {
	DurationMs   int            `json:"duration_ms"`
	URL          string         `json:"url"`
	Thumb        string         `json:"thumb"`
	Views        string         `json:"views"`
	Available    bool           `json:"available"`
	Reason       string         `json:"reason"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	PlaybackType VideoType      `json:"playback_type"`
	Variants     []VideoVariant `json:"variants"`
}

type Gif struct {
	URL   string `json:"url"`
	Thumb string `json:"thumb"`
}

type Photo struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Poll struct {
	Options []string `json:"options"`
	Values  []int    `json:"values"`
	Votes   int      `json:"votes"`
	Leader  int      `json:"leader"`
	Status  string   `json:"status"`
}

type CardKind string

const (
	CardSummary      CardKind = "summary"
	CardSummaryLarge CardKind = "summary_large_image"
	CardPlayer       CardKind = "player"
	CardApp          CardKind = "app"
	CardAmplify      CardKind = "amplify"
	CardUnified      CardKind = "unified_card"
)

type Card struct {
	Kind        CardKind `json:"kind"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Destination string   `json:"destination"`
	Description string   `json:"description"`
	Image       string   `json:"image"`
	Video       *Video   `json:"video,omitempty"`
}

type TweetStats struct {
	Replies  int `json:"replies"`
	Retweets int `json:"retweets"`
	Likes    int `json:"likes"`
	Quotes   int `json:"quotes"`
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Fullname  string    `json:"fullname"`
	Location  string    `json:"location"`
	Website   string    `json:"website"`
	Bio       string    `json:"bio"`
	UserPic   string    `json:"userpic"`
	Banner    string    `json:"banner"`
	Following int       `json:"following"`
	Followers int       `json:"followers"`
	Tweets    int       `json:"tweets"`
	Likes     int       `json:"likes"`
	Media     int       `json:"media"`
	Verified  bool      `json:"verified"`
	Protected bool      `json:"protected"`
	Suspended bool      `json:"suspended"`
	JoinDate  time.Time `json:"join_date"`
}

type Tweet struct {
	ID          int64      `json:"id"`
	ThreadID    int64      `json:"thread_id"`
	ReplyID     int64      `json:"reply_id"`
	User        User       `json:"user"`
	Text        string     `json:"text"`
	Time        time.Time  `json:"time"`
	Reply       []string   `json:"reply"`
	Pinned      bool       `json:"pinned"`
	HasThread   bool       `json:"has_thread"`
	Available   bool       `json:"available"`
	Tombstone   string     `json:"tombstone"`
	Location    string     `json:"location"`
	Stats       TweetStats `json:"stats"`
	Retweet     *Tweet     `json:"retweet,omitempty"`
	Quote       *Tweet     `json:"quote,omitempty"`
	Card        *Card      `json:"card,omitempty"`
	Poll        *Poll      `json:"poll,omitempty"`
	Gif         *Gif       `json:"gif,omitempty"`
	Video       *Video     `json:"video,omitempty"`
	Photos      []Photo    `json:"photos"`
	MediaTags   []User     `json:"media_tags"`
	Attribution *User      `json:"attribution,omitempty"`
}

type Timeline struct {
	Content   []Tweet `json:"content"`
	Top       string  `json:"top"`
	Bottom    string  `json:"bottom"`
	Beginning bool    `json:"beginning"`
}

type Profile struct {
	User      User     `json:"user"`
	PhotoRail []Photo  `json:"photo_rail"`
	Pinned    *Tweet   `json:"pinned,omitempty"`
	Tweets    Timeline `json:"tweets"`
}

type Chain struct {
	Content []Tweet `json:"content"`
	HasMore bool    `json:"has_more"`
	Cursor  string  `json:"cursor"`
}

type Conversation struct {
	Tweet   Tweet   `json:"tweet"`
	Before  Chain   `json:"before"`
	After   Chain   `json:"after"`
	Replies []Chain `json:"replies"`
}

type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Description string `json:"description"`
	Members     int    `json:"members"`
	Banner      string `json:"banner"`
}

type SearchResult struct {
	Tweets Timeline `json:"tweets"`
	Users  []User   `json:"users"`
}

// MediaType represents what kind of media a tweet contains
type MediaType string

const (
	MediaTypePhoto   MediaType = "photo"
	MediaTypeVideo   MediaType = "video"
	MediaTypeGif     MediaType = "gif"
	MediaTypeSticker MediaType = "sticker"
	MediaTypeNone    MediaType = "none"
)

// MediaAttachment represents a downloadable media item in a tweet
type MediaAttachment struct {
	Type        MediaType `json:"type"`
	URL         string    `json:"url"`
	DirectURL   string    `json:"direct_url"`
	ThumbURL    string    `json:"thumb_url"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	Duration    int       `json:"duration_ms,omitempty"`
	Bitrate     int       `json:"bitrate,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
}

// GetMediaAttachments extracts all media from a tweet as downloadable attachments
func (t *Tweet) GetMediaAttachments() []MediaAttachment {
	var attachments []MediaAttachment

	for _, photo := range t.Photos {
		// Get the highest quality image URL (orig size)
		directURL := photo.URL
		if !containsSuffix(directURL, "orig") {
			if idx := lastIndexByte(directURL, '.'); idx > 0 {
				directURL = directURL + "?name=orig"
			}
		}
		attachments = append(attachments, MediaAttachment{
			Type:        MediaTypePhoto,
			URL:         photo.URL,
			DirectURL:   directURL,
			ThumbURL:    photo.URL + "?name=small",
			Width:       photo.Width,
			Height:      photo.Height,
			ContentType: "image/jpeg",
		})
	}

	if t.Video != nil && t.Video.Available {
		bestVariant := getBestVideoVariant(t.Video.Variants)
		directURL := ""
		contentType := ""
		bitrate := 0
		if bestVariant != nil {
			directURL = bestVariant.URL
			contentType = string(bestVariant.ContentType)
			bitrate = bestVariant.Bitrate
		}
		attachments = append(attachments, MediaAttachment{
			Type:        MediaTypeVideo,
			URL:         t.Video.URL,
			DirectURL:   directURL,
			ThumbURL:    t.Video.Thumb,
			Duration:    t.Video.DurationMs,
			Bitrate:     bitrate,
			ContentType: contentType,
		})
	}

	if t.Gif != nil {
		attachments = append(attachments, MediaAttachment{
			Type:        MediaTypeGif,
			URL:         t.Gif.URL,
			DirectURL:   t.Gif.URL,
			ThumbURL:    t.Gif.Thumb,
			ContentType: "video/mp4",
		})
	}

	return attachments
}

// GetMediaTypes returns a list of media types in the tweet
func (t *Tweet) GetMediaTypes() []MediaType {
	var types []MediaType
	if len(t.Photos) > 0 {
		types = append(types, MediaTypePhoto)
	}
	if t.Video != nil && t.Video.Available {
		types = append(types, MediaTypeVideo)
	}
	if t.Gif != nil {
		types = append(types, MediaTypeGif)
	}
	return types
}

func getBestVideoVariant(variants []VideoVariant) *VideoVariant {
	var best *VideoVariant
	for i := range variants {
		v := &variants[i]
		if v.ContentType == VideoTypeMP4 {
			if best == nil || v.Bitrate > best.Bitrate {
				best = v
			}
		}
	}
	if best == nil && len(variants) > 0 {
		best = &variants[0]
	}
	return best
}

func containsSuffix(url, suffix string) bool {
	return len(url) > len(suffix) && url[len(url)-len(suffix):] == suffix
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
