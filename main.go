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

	// Initialize Twitter client
	client := twitter.NewClient(cfg.MinTokens, cfg.EnableDebug)
	client.InitTokenPool()

	// Set up HTTP handlers
	handler, err := handlers.New(cfg, client, redisCache)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	fmt.Printf("Nitter (Go) running on http://%s\n", addr)
	fmt.Printf("RSS feeds enabled: %v\n", cfg.EnableRSS)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
