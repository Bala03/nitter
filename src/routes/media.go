package main

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const (
	twimg     = "twimg.com"
	https     = "https://"
	m3u8Mime  = "application/vnd.apple.mpegurl"
	maxAge    = "max-age=604800"
)

func safeFetch(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func proxyMedia(w http.ResponseWriter, r *http.Request, url string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch media: %s", resp.Status)
	}

	hasher := fnv.New32a()
	hasher.Write([]byte(url))
	hashed := fmt.Sprintf("%x", hasher.Sum32())

	if r.Header.Get("If-None-Match") == hashed {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	w.Header().Set("Cache-Control", maxAge)
	w.Header().Set("ETag", hashed)

	_, err = io.Copy(w, resp.Body)
	return err
}

func decode(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func createMediaRouter(r *mux.Router) {
	r.HandleFunc("/pic/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}).Methods("GET")

	r.HandleFunc("/pic/orig/{enc}/{url}", func(w http.ResponseWriter, r *http.Request) {
		url, err := decode(mux.Vars(r)["url"])
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		if !strings.Contains(url, twimg) {
			url = twimg + url
		}
		if !strings.HasPrefix(url, https) {
			url = https + url
		}
		url += "?name=orig"

		if err := proxyMedia(w, r, url); err != nil {
			http.Error(w, "Failed to proxy media", http.StatusNotFound)
		}
	}).Methods("GET")

	r.HandleFunc("/pic/{enc}/{url}", func(w http.ResponseWriter, r *http.Request) {
		url, err := decode(mux.Vars(r)["url"])
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		if !strings.Contains(url, twimg) {
			url = twimg + url
		}
		if !strings.HasPrefix(url, https) {
			url = https + url
		}

		if err := proxyMedia(w, r, url); err != nil {
			http.Error(w, "Failed to proxy media", http.StatusNotFound)
		}
	}).Methods("GET")

	r.HandleFunc("/video/{enc}/{url}/{sig}", func(w http.ResponseWriter, r *http.Request) {
		url, err := decode(mux.Vars(r)["url"])
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		if !strings.HasPrefix(url, "http") {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		if getHmac(url) != mux.Vars(r)["sig"] {
			http.Error(w, "Failed to verify signature", http.StatusForbidden)
			return
		}

		if strings.Contains(url, ".mp4") || strings.Contains(url, ".ts") || strings.Contains(url, ".m4s") {
			if err := proxyMedia(w, r, url); err != nil {
				http.Error(w, "Failed to proxy media", http.StatusNotFound)
			}
			return
		}

		if strings.Contains(url, ".vmap") {
			m3u8, err := getM3u8Url(safeFetch(url))
			if err != nil {
				http.Error(w, "Failed to fetch media", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", m3u8Mime)
			w.Write([]byte(m3u8))
			return
		}

		if strings.Contains(url, ".m3u8") {
			vid, err := safeFetch(url)
			if err != nil {
				http.Error(w, "Failed to fetch media", http.StatusNotFound)
				return
			}
			content := proxifyVideo(vid, cookiePref(proxyVideos))
			w.Header().Set("Content-Type", m3u8Mime)
			w.Write([]byte(content))
			return
		}

		http.Error(w, "Unsupported media type", http.StatusBadRequest)
	}).Methods("GET")
}
