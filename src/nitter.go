package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	Address    string
	StaticDir  string
	EnableRss  bool
	EnableDebug bool
}

var cfg Config

func initConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	cfg = Config{
		Port:       getEnv("PORT", "8080"),
		Address:    getEnv("ADDRESS", "0.0.0.0"),
		StaticDir:  getEnv("STATIC_DIR", "./static"),
		EnableRss:  getEnvAsBool("ENABLE_RSS", true),
		EnableDebug: getEnvAsBool("ENABLE_DEBUG", false),
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func main() {
	initConfig()

	r := mux.NewRouter()

	r.HandleFunc("/", renderMain).Methods("GET")
	r.HandleFunc("/about", renderAbout).Methods("GET")
	r.HandleFunc("/explore", redirectToAbout).Methods("GET")
	r.HandleFunc("/help", redirectToAbout).Methods("GET")
	r.HandleFunc("/i/redirect", handleRedirect).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(handleNotFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(handleMethodNotAllowed)

	http.Handle("/", r)

	log.Printf("Starting server at %s:%s\n", cfg.Address, cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Address, cfg.Port), nil))
}

func renderMain(w http.ResponseWriter, r *http.Request) {
	// Implement the main rendering logic here
}

func renderAbout(w http.ResponseWriter, r *http.Request) {
	// Implement the about page rendering logic here
}

func redirectToAbout(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/about", http.StatusFound)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "URL parameter is missing", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Page not found", http.StatusNotFound)
}

func handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
