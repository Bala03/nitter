package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	UserPic  string `json:"userPic"`
	Banner   string `json:"banner"`
}

type Tweet struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	User     User   `json:"user"`
	Photos   []string `json:"photos"`
	Videos   []Video  `json:"videos"`
}

type Video struct {
	URL    string `json:"url"`
	Thumb  string `json:"thumb"`
	Views  int    `json:"views"`
	Title  string `json:"title"`
}

func fetchUser(username string) (User, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", username)
	resp, err := http.Get(url)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}

	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func fetchTweets(userID string) ([]Tweet, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets", userID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tweets []Tweet
	err = json.Unmarshal(body, &tweets)
	if err != nil {
		return nil, err
	}

	return tweets, nil
}

func fetchMedia(tweetID string) ([]string, []Video, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/tweets/%s", tweetID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var tweet Tweet
	err = json.Unmarshal(body, &tweet)
	if err != nil {
		return nil, nil, err
	}

	return tweet.Photos, tweet.Videos, nil
}

func main() {
	username := "example_user"
	user, err := fetchUser(username)
	if err != nil {
		fmt.Println("Error fetching user:", err)
		return
	}

	tweets, err := fetchTweets(user.ID)
	if err != nil {
		fmt.Println("Error fetching tweets:", err)
		return
	}

	for _, tweet := range tweets {
		photos, videos, err := fetchMedia(tweet.ID)
		if err != nil {
			fmt.Println("Error fetching media:", err)
			continue
		}

		fmt.Printf("Tweet: %s\n", tweet.Text)
		fmt.Printf("Photos: %v\n", photos)
		fmt.Printf("Videos: %v\n", videos)
	}
}
