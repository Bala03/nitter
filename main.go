package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zedeus/nitter/cache"
	"github.com/zedeus/nitter/config"
	"github.com/zedeus/nitter/handlers"
	"github.com/zedeus/nitter/twitter"
)

func main() {
	configPath := "nitter.conf"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Redis
	redisCache, err := cache.NewRedisCache(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, cfg.RedisConns)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	fmt.Printf("Connected to Redis at %s:%d\n", cfg.RedisHost, cfg.RedisPort)

	// Initialize session store
	sessionStore := twitter.NewSessionStore(cfg.SessionFile)
	activeSessions := sessionStore.GetActive()
	fmt.Printf("Loaded %d session(s) from %s (%d active)\n",
		len(sessionStore.GetAll()), cfg.SessionFile, len(activeSessions))

	// Initialize Twitter client
	client := twitter.NewClient(cfg.MinTokens, cfg.EnableDebug)
	client.SetSessionStore(sessionStore)
	client.InitTokenPool()

	// Set up HTTP handlers
	handler, err := handlers.New(cfg, client, redisCache)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	handler.RegisterAdminRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	fmt.Printf("Nitter (Go) running on http://%s\n", addr)
	fmt.Printf("RSS feeds enabled: %v\n", cfg.EnableRSS)
	if cfg.AdminPassword != "" {
		fmt.Printf("Admin panel enabled at /admin\n")
	} else {
		fmt.Printf("Admin panel disabled (set adminPassword in config to enable)\n")
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
