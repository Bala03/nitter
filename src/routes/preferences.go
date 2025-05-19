package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Preferences struct {
	Theme       string `json:"theme"`
	Language    string `json:"language"`
	ShowImages  bool   `json:"show_images"`
	ShowVideos  bool   `json:"show_videos"`
}

var (
	preferences = make(map[string]Preferences)
	mu          sync.Mutex
)

func savePreferences(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var prefs Preferences
	err := json.NewDecoder(r.Body).Decode(&prefs)
	if err != nil {
		http.Error(w, "Invalid preferences data", http.StatusBadRequest)
		return
	}

	mu.Lock()
	preferences[username] = prefs
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func resetPreferences(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	delete(preferences, username)
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func getPreferences(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	prefs, exists := preferences[username]
	mu.Unlock()

	if !exists {
		http.Error(w, "Preferences not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

func createPreferencesRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/preferences/save", savePreferences)
	mux.HandleFunc("/preferences/reset", resetPreferences)
	mux.HandleFunc("/preferences/get", getPreferences)
	return mux
}
