package rss

import (
	"encoding/xml"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/zedeus/nitter/twitter"
)

// RSS XML structures with media extensions
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	DC      string   `xml:"xmlns:dc,attr"`
	Content string   `xml:"xmlns:content,attr"`
	Atom    string   `xml:"xmlns:atom,attr"`
	Media   string   `xml:"xmlns:media,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	AtomLink      *AtomLink `xml:"atom:link,omitempty"`
	Image         *Image    `xml:"image,omitempty"`
	Items         []Item    `xml:"item"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type Image struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

type Item struct {
	Title         string       `xml:"title"`
	Link          string       `xml:"link"`
	Description   CDATA        `xml:"description"`
	Author        string       `xml:"dc:creator"`
	PubDate       string       `xml:"pubDate"`
	GUID          GUID         `xml:"guid"`
	Enclosures    []Enclosure  `xml:"enclosure,omitempty"`
	MediaContents []MediaContent `xml:"media:content,omitempty"`
	MediaGroup    *MediaGroup  `xml:"media:group,omitempty"`
}

type GUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

type CDATA struct {
	Value string `xml:",cdata"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length string `xml:"length,attr,omitempty"`
}

type MediaContent struct {
	URL      string `xml:"url,attr"`
	Type     string `xml:"type,attr,omitempty"`
	Medium   string `xml:"medium,attr,omitempty"`
	Width    string `xml:"width,attr,omitempty"`
	Height   string `xml:"height,attr,omitempty"`
	Duration string `xml:"duration,attr,omitempty"`
	Bitrate  string `xml:"bitrate,attr,omitempty"`
}

type MediaGroup struct {
	Contents []MediaContent `xml:"media:content,omitempty"`
}

// GenerateUserFeed creates an RSS feed for a user's timeline
func GenerateUserFeed(user *twitter.User, tweets []twitter.Tweet, baseURL string) (string, error) {
	rssURL := fmt.Sprintf("%s/%s/rss", baseURL, user.Username)
	profileURL := fmt.Sprintf("%s/%s", baseURL, user.Username)

	channel := Channel{
		Title:       fmt.Sprintf("%s / @%s", user.Fullname, user.Username),
		Link:        profileURL,
		Description: user.Bio,
		Language:    "en",
		AtomLink: &AtomLink{
			Href: rssURL,
			Rel:  "self",
			Type: "application/rss+xml",
		},
	}

	if user.UserPic != "" {
		channel.Image = &Image{
			URL:   user.UserPic,
			Title: user.Fullname,
			Link:  profileURL,
		}
	}

	if len(tweets) > 0 {
		channel.LastBuildDate = tweets[0].Time.Format(time.RFC1123Z)
	}

	for _, tweet := range tweets {
		item := tweetToItem(tweet, baseURL)
		channel.Items = append(channel.Items, item)
	}

	return marshalRSS(channel)
}

// GenerateSearchFeed creates an RSS feed for search results
func GenerateSearchFeed(query string, tweets []twitter.Tweet, baseURL string) (string, error) {
	channel := Channel{
		Title:       fmt.Sprintf("Search: %s", query),
		Link:        fmt.Sprintf("%s/search?q=%s", baseURL, query),
		Description: fmt.Sprintf("Twitter search results for: %s", query),
		Language:    "en",
	}

	if len(tweets) > 0 {
		channel.LastBuildDate = tweets[0].Time.Format(time.RFC1123Z)
	}

	for _, tweet := range tweets {
		item := tweetToItem(tweet, baseURL)
		channel.Items = append(channel.Items, item)
	}

	return marshalRSS(channel)
}

func tweetToItem(tweet twitter.Tweet, baseURL string) Item {
	tweetURL := fmt.Sprintf("%s/%s/status/%d", baseURL, tweet.User.Username, tweet.ID)
	twitterURL := fmt.Sprintf("https://twitter.com/%s/status/%d", tweet.User.Username, tweet.ID)

	title := buildTitle(tweet)
	description := buildDescription(tweet, baseURL)

	item := Item{
		Title:  title,
		Link:   tweetURL,
		Author: fmt.Sprintf("@%s", tweet.User.Username),
		GUID: GUID{
			Value:       twitterURL,
			IsPermaLink: true,
		},
		PubDate: tweet.Time.Format(time.RFC1123Z),
		Description: CDATA{
			Value: description,
		},
	}

	// Add media enclosures and media:content elements
	attachments := tweet.GetMediaAttachments()
	if len(attachments) > 0 {
		var mediaContents []MediaContent

		for _, att := range attachments {
			// Add enclosure for direct download
			if att.DirectURL != "" {
				enc := Enclosure{
					URL:  att.DirectURL,
					Type: att.ContentType,
				}
				if att.Bitrate > 0 {
					enc.Length = strconv.Itoa(att.Bitrate)
				}
				item.Enclosures = append(item.Enclosures, enc)
			}

			// Add media:content with full metadata
			mc := MediaContent{
				URL: att.DirectURL,
			}
			if att.ContentType != "" {
				mc.Type = att.ContentType
			}
			switch att.Type {
			case twitter.MediaTypePhoto:
				mc.Medium = "image"
			case twitter.MediaTypeVideo:
				mc.Medium = "video"
			case twitter.MediaTypeGif:
				mc.Medium = "video"
			}
			if att.Width > 0 {
				mc.Width = strconv.Itoa(att.Width)
			}
			if att.Height > 0 {
				mc.Height = strconv.Itoa(att.Height)
			}
			if att.Duration > 0 {
				mc.Duration = strconv.Itoa(att.Duration / 1000)
			}
			if att.Bitrate > 0 {
				mc.Bitrate = strconv.Itoa(att.Bitrate / 1000)
			}
			mediaContents = append(mediaContents, mc)
		}

		if len(mediaContents) > 1 {
			item.MediaGroup = &MediaGroup{Contents: mediaContents}
		} else if len(mediaContents) == 1 {
			item.MediaContents = mediaContents
		}
	}

	return item
}

func buildTitle(tweet twitter.Tweet) string {
	text := tweet.Text
	if len(text) > 140 {
		text = text[:137] + "..."
	}
	// Remove newlines for title
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

func buildDescription(tweet twitter.Tweet, baseURL string) string {
	var sb strings.Builder

	// Tweet text
	sb.WriteString("<p>")
	sb.WriteString(html.EscapeString(tweet.Text))
	sb.WriteString("</p>")

	// Media info section
	attachments := tweet.GetMediaAttachments()
	if len(attachments) > 0 {
		sb.WriteString(`<div class="media-attachments">`)

		for _, att := range attachments {
			switch att.Type {
			case twitter.MediaTypePhoto:
				sb.WriteString(fmt.Sprintf(
					`<p><strong>[Image]</strong> <a href="%s">Direct Download (%dx%d)</a></p>`,
					html.EscapeString(att.DirectURL),
					att.Width, att.Height,
				))
				sb.WriteString(fmt.Sprintf(
					`<img src="%s" width="%d" height="%d" alt="Tweet image" />`,
					html.EscapeString(att.ThumbURL),
					att.Width, att.Height,
				))

			case twitter.MediaTypeVideo:
				durationStr := ""
				if att.Duration > 0 {
					seconds := att.Duration / 1000
					minutes := seconds / 60
					secs := seconds % 60
					durationStr = fmt.Sprintf(" Duration: %d:%02d", minutes, secs)
				}
				bitrateStr := ""
				if att.Bitrate > 0 {
					bitrateStr = fmt.Sprintf(" Bitrate: %d kbps", att.Bitrate/1000)
				}
				sb.WriteString(fmt.Sprintf(
					`<p><strong>[Video]</strong> <a href="%s">Direct Download (MP4)</a>%s%s</p>`,
					html.EscapeString(att.DirectURL),
					durationStr, bitrateStr,
				))
				if att.ThumbURL != "" {
					sb.WriteString(fmt.Sprintf(
						`<img src="%s" alt="Video thumbnail" />`,
						html.EscapeString(att.ThumbURL),
					))
				}

			case twitter.MediaTypeGif:
				sb.WriteString(fmt.Sprintf(
					`<p><strong>[GIF]</strong> <a href="%s">Direct Download (MP4)</a></p>`,
					html.EscapeString(att.DirectURL),
				))
				if att.ThumbURL != "" {
					sb.WriteString(fmt.Sprintf(
						`<img src="%s" alt="GIF thumbnail" />`,
						html.EscapeString(att.ThumbURL),
					))
				}

			case twitter.MediaTypeSticker:
				sb.WriteString(fmt.Sprintf(
					`<p><strong>[Sticker]</strong> <a href="%s">Direct Download</a></p>`,
					html.EscapeString(att.DirectURL),
				))
			}
		}
		sb.WriteString(`</div>`)
	}

	// Video variants (all quality options)
	if tweet.Video != nil && len(tweet.Video.Variants) > 0 {
		sb.WriteString(`<div class="video-variants">`)
		sb.WriteString(`<p><strong>All video qualities:</strong></p><ul>`)
		for _, v := range tweet.Video.Variants {
			if v.ContentType == twitter.VideoTypeMP4 {
				label := "MP4"
				if v.Bitrate > 0 {
					label = fmt.Sprintf("MP4 %d kbps", v.Bitrate/1000)
				}
				sb.WriteString(fmt.Sprintf(
					`<li><a href="%s">%s</a></li>`,
					html.EscapeString(v.URL), label,
				))
			}
		}
		sb.WriteString(`</ul></div>`)
	}

	// Quote tweet
	if tweet.Quote != nil {
		sb.WriteString(`<blockquote>`)
		sb.WriteString(fmt.Sprintf(`<p><strong>@%s:</strong> %s</p>`,
			html.EscapeString(tweet.Quote.User.Username),
			html.EscapeString(tweet.Quote.Text),
		))
		quoteAttachments := tweet.Quote.GetMediaAttachments()
		for _, att := range quoteAttachments {
			switch att.Type {
			case twitter.MediaTypePhoto:
				sb.WriteString(fmt.Sprintf(
					`<p>[Quoted Image] <a href="%s">Download</a></p>`,
					html.EscapeString(att.DirectURL),
				))
			case twitter.MediaTypeVideo:
				sb.WriteString(fmt.Sprintf(
					`<p>[Quoted Video] <a href="%s">Download</a></p>`,
					html.EscapeString(att.DirectURL),
				))
			case twitter.MediaTypeGif:
				sb.WriteString(fmt.Sprintf(
					`<p>[Quoted GIF] <a href="%s">Download</a></p>`,
					html.EscapeString(att.DirectURL),
				))
			}
		}
		sb.WriteString(`</blockquote>`)
	}

	// Stats
	sb.WriteString(fmt.Sprintf(
		`<p class="stats">♻️ %d | ❤️ %d | 💬 %d</p>`,
		tweet.Stats.Retweets, tweet.Stats.Likes, tweet.Stats.Replies,
	))

	return sb.String()
}

func marshalRSS(channel Channel) (string, error) {
	rss := RSS{
		Version: "2.0",
		DC:      "http://purl.org/dc/elements/1.1/",
		Content: "http://purl.org/rss/1.0/modules/content/",
		Atom:    "http://www.w3.org/2005/Atom",
		Media:   "http://search.yahoo.com/mrss/",
		Channel: channel,
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}
