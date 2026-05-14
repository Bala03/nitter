package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `[Server]
hostname = "test.nitter.net"
title = "Test Nitter"
address = "127.0.0.1"
port = 9090
https = true
httpMaxConnections = 50
staticDir = "./public"

[Cache]
listMinutes = 60
rssMinutes = 5
redisHost = "redis.local"
redisPort = 6380
redisPassword = "secret"
redisConnections = 10
redisMaxConnections = 50

[Config]
hmacKey = "mykey"
base64Media = true
tokenCount = 5
enableRSS = true
enableDebug = true
proxy = "http://proxy.local:8080"
proxyAuth = "user:pass"
`
	tmpfile, err := os.CreateTemp("", "nitter-*.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Server
	if cfg.Hostname != "test.nitter.net" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "test.nitter.net")
	}
	if cfg.Title != "Test Nitter" {
		t.Errorf("Title = %q, want %q", cfg.Title, "Test Nitter")
	}
	if cfg.Address != "127.0.0.1" {
		t.Errorf("Address = %q, want %q", cfg.Address, "127.0.0.1")
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}
	if !cfg.UseHTTPS {
		t.Error("UseHTTPS should be true")
	}

	// Cache
	if cfg.RedisHost != "redis.local" {
		t.Errorf("RedisHost = %q, want %q", cfg.RedisHost, "redis.local")
	}
	if cfg.RedisPort != 6380 {
		t.Errorf("RedisPort = %d, want %d", cfg.RedisPort, 6380)
	}
	if cfg.RedisPassword != "secret" {
		t.Errorf("RedisPassword = %q, want %q", cfg.RedisPassword, "secret")
	}
	if cfg.RSSCacheTime != 5 {
		t.Errorf("RSSCacheTime = %d, want %d", cfg.RSSCacheTime, 5)
	}

	// Config
	if cfg.HMACKey != "mykey" {
		t.Errorf("HMACKey = %q, want %q", cfg.HMACKey, "mykey")
	}
	if !cfg.Base64Media {
		t.Error("Base64Media should be true")
	}
	if cfg.MinTokens != 5 {
		t.Errorf("MinTokens = %d, want %d", cfg.MinTokens, 5)
	}
	if !cfg.EnableRSS {
		t.Error("EnableRSS should be true")
	}
	if !cfg.EnableDebug {
		t.Error("EnableDebug should be true")
	}
	if cfg.Proxy != "http://proxy.local:8080" {
		t.Errorf("Proxy = %q, want %q", cfg.Proxy, "http://proxy.local:8080")
	}
}

func TestBaseURL(t *testing.T) {
	cfg := &Config{Hostname: "example.com", UseHTTPS: true}
	if cfg.BaseURL() != "https://example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL(), "https://example.com")
	}

	cfg.UseHTTPS = false
	if cfg.BaseURL() != "http://example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL(), "http://example.com")
	}
}
